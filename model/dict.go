/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package model

import "gorm.io/gorm"

type Dict struct {
	Group uint32 `json:"group" gorm:"primaryKey;comment:组"`
	Key   string `json:"key" gorm:"primaryKey;comment:键"`
	Value string `json:"value" gorm:"comment:值"`
	Type  uint32 `json:"type" gorm:"comment:类型"`
	Seq   uint32 `json:"seq" gorm:"comment:排序"`
}


func DictGetValue(db *gorm.DB, typ int, key string) (string, error) {
	var value string
	err := db.Table(`dict`).Select(`value`).Where(`type = ? AND key=?`, typ, key).Scan(&value).Error
	if err != nil {
		return "", err
	}
	return value, nil
}

func DictSetValue(db *gorm.DB, typ int, key, value string) error {
	return db.Table(`dict`).Where(`type = ? AND key=?`, typ, key).UpdateColumn("value", value).Error
}
