package utils

// SlicePage 通用分页切片，返回分页后的新切片和原始总数
func SlicePage[T any](slice []T, pageNo, pageSize int) ([]T, int) {
	total := len(slice)
	if total == 0 {
		return slice, 0
	}
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (pageNo - 1) * pageSize
	if start >= total {
		var zero []T
		return zero, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	paged := make([]T, end-start)
	copy(paged, slice[start:end])
	return paged, total
}
