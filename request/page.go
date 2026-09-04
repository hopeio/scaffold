package param

// Pagination bounds shared by every list endpoint.
//
// gormx.PaginationExpr emits no LIMIT when pageSize is 0, which returns the
// whole table; a client can also pass a huge pageSize and stall the database.
// Every list request goes through Clamp/ClampInt first.
const (
	// DefaultPageSize is used when the client omits pageSize.
	DefaultPageSize uint32 = 20
	// MaxPageSize is the upper bound of a single page.
	MaxPageSize uint32 = 200
)

// Clamp normalizes pagination parameters; the result can be passed straight to
// gormx.PaginationExpr.
func Clamp(pageNo, pageSize uint32) (uint32, uint32) {
	if pageNo == 0 {
		pageNo = 1
	}
	switch {
	case pageSize == 0:
		pageSize = DefaultPageSize
	case pageSize > MaxPageSize:
		pageSize = MaxPageSize
	}
	return pageNo, pageSize
}

// ClampInt is the int flavour of Clamp, for DAOs that pass ints around.
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
