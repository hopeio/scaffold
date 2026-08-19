// Package page 统一分页参数的取值范围。
//
// gormx.PaginationExpr 在 pageSize 为 0 时不生成 LIMIT，等于整表返回；
// 反过来客户端也能塞一个极大的 pageSize 把库拖垮。所有列表接口先过这里。
package page

const (
	// Default 客户端没给 pageSize 时的默认值。
	Default uint32 = 20
	// Max 单页上限。
	Max uint32 = 200
)

// Clamp 归一化分页参数，返回值可直接交给 gormx.PaginationExpr。
func Clamp(pageNo, pageSize uint32) (uint32, uint32) {
	if pageNo == 0 {
		pageNo = 1
	}
	switch {
	case pageSize == 0:
		pageSize = Default
	case pageSize > Max:
		pageSize = Max
	}
	return pageNo, pageSize
}

// ClampInt 是 Clamp 的 int 版本，给按 int 传参的 DAO 用。
func ClampInt(pageNo, pageSize int) (int, int) {
	if pageNo < 0 {
		pageNo = 0
	}
	if pageSize < 0 {
		pageSize = 0
	}
	no, size := Clamp(uint32(pageNo), uint32(pageSize))
	return int(no), int(size)
}
