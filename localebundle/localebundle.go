/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

// Package localebundle 管理多语言文件基线：目录下 *.json 是基线词条，
// 支持进程内缓存与 fsnotify 热更新。机制层：目录由调用方传入，不依赖 global。
package localebundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hopeio/gox/log"
	pathx "github.com/hopeio/gox/os/fs/path"
	"go.uber.org/zap"
)

var (
	lock           sync.RWMutex
	localeMessages = map[string]map[string]string{}
	// 文件层最后加载时间：与 DB 最新变更取大者构成 bundle 版本号。
	fileVersion int64
	loadOnce    sync.Once
	loadedDir   string
)

// Load 按目录加载（进程内只加载一次，重复调用返回缓存）。
func Load(dir string) {
	loadOnce.Do(func() {
		loadedDir = dir
		files, err := os.ReadDir(dir)
		if err != nil {
			log.Errorw("read locale dir failed", zap.Error(err), zap.String("dir", dir))
			return
		}
		lock.Lock()
		for _, file := range files {
			messages := loadLocaleFile(filepath.Join(dir, file.Name()))
			if len(messages) == 0 {
				continue
			}
			localeMessages[pathx.FileNoExt(file.Name())] = messages
		}
		fileVersion = time.Now().UnixMilli()
		lock.Unlock()
		watchLocaleDir(dir)
	})
}

// Has 该语言文件基线是否存在。
func Has(locale string) bool {
	lock.RLock()
	defer lock.RUnlock()
	_, ok := localeMessages[locale]
	return ok
}

// Version 文件层版本（最后加载时间的毫秒时间戳）。
func Version() int64 {
	lock.RLock()
	defer lock.RUnlock()
	return fileVersion
}

// Messages 文件基线；返回拷贝，调用方可随意改。
func Messages(locale string) map[string]string {
	lock.RLock()
	defer lock.RUnlock()
	src := localeMessages[locale]
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func watchLocaleDir(dir string) {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorw("fsnotify.NewWatcher error", zap.Error(err), zap.String("dir", dir))
		return
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Errorw("filepath.Abs error", zap.Error(err), zap.String("dir", dir))
		return
	}
	if err = watch.Add(absDir); err != nil {
		log.Errorw("watch.Add error", zap.Error(err), zap.String("dir", dir))
		return
	}
	go func() {
		defer watch.Close()
		for {
			select {
			case event, ok := <-watch.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					messages := loadLocaleFile(event.Name)
					if len(messages) == 0 {
						continue
					}
					lock.Lock()
					localeMessages[pathx.FileNoExt(filepath.Base(event.Name))] = messages
					fileVersion = time.Now().UnixMilli()
					lock.Unlock()
					log.Infow("locale file reloaded", zap.String("file", event.Name))
				}
			case err, ok := <-watch.Errors:
				if !ok {
					return
				}
				log.Errorw("locale file watch error", zap.Error(err))
			}
		}
	}()
}

func loadLocaleFile(file string) map[string]string {
	path := filepath.Clean(file)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	messages := make(map[string]string)
	if err = json.Unmarshal(data, &messages); err != nil {
		return nil
	}
	return messages
}
