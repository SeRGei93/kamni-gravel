package handler

import (
	"log"
	"net/http"
	"strconv"
)

// Параметры серверной пагинации. Размер страницы настраивается пользователем,
// но ограничен диапазоном [50, 100] (требование продукта).
const (
	DefaultPageSize = 50
	MinPageSize     = 50
	MaxPageSize     = 100
)

// PageParams содержит разобранные и нормализованные параметры пагинации.
type PageParams struct {
	Page     int // 1-based номер страницы
	PageSize int // размер страницы (50..100)
	Limit    int // = PageSize, для SQL LIMIT
	Offset   int // = (Page-1)*PageSize, для SQL OFFSET
}

// TotalPages возвращает количество страниц для заданного общего числа элементов.
func (p PageParams) TotalPages(total int) int {
	if p.PageSize <= 0 || total <= 0 {
		return 0
	}
	return (total + p.PageSize - 1) / p.PageSize
}

// ParsePageParams разбирает query-параметры `page` и `page_size`.
// page по умолчанию 1 (минимум 1); page_size по умолчанию 50 и зажимается в [50, 100].
// Некорректные/пустые значения заменяются значениями по умолчанию.
func ParsePageParams(r *http.Request) PageParams {
	rawPage := r.URL.Query().Get("page")
	rawSize := r.URL.Query().Get("page_size")

	page := 1
	if rawPage != "" {
		if p, err := strconv.Atoi(rawPage); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := DefaultPageSize
	if rawSize != "" {
		if s, err := strconv.Atoi(rawSize); err == nil {
			pageSize = s
		}
	}
	if pageSize < MinPageSize {
		pageSize = MinPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	pp := PageParams{
		Page:     page,
		PageSize: pageSize,
		Limit:    pageSize,
		Offset:   (page - 1) * pageSize,
	}

	log.Printf(
		"DEBUG Pagination params parsed: path=%s raw_page=%q raw_page_size=%q page=%d page_size=%d limit=%d offset=%d",
		r.URL.Path, rawPage, rawSize, pp.Page, pp.PageSize, pp.Limit, pp.Offset,
	)

	return pp
}
