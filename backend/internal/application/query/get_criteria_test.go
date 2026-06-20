package query

import (
	"context"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

type criteriaRepoFake struct {
	items        []*entity.Criteria
	total        int
	gotLimit     int
	gotOffset    int
	gotTypePtr   *valueobject.CriteriaType
	listPagedHit bool
}

func (r *criteriaRepoFake) Create(ctx context.Context, c *entity.Criteria) error { return nil }
func (r *criteriaRepoFake) Update(ctx context.Context, c *entity.Criteria) error { return nil }
func (r *criteriaRepoFake) Delete(ctx context.Context, id uint) error            { return nil }
func (r *criteriaRepoFake) FindByID(ctx context.Context, id uint) (*entity.Criteria, error) {
	return nil, nil
}
func (r *criteriaRepoFake) FindAll(ctx context.Context) ([]*entity.Criteria, error) { return nil, nil }
func (r *criteriaRepoFake) FindByType(ctx context.Context, ct valueobject.CriteriaType) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *criteriaRepoFake) FindByGift(ctx context.Context, giftID uint) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *criteriaRepoFake) FindByResult(ctx context.Context, resultID uint) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *criteriaRepoFake) ListPaged(ctx context.Context, ct *valueobject.CriteriaType, limit, offset int) ([]*entity.Criteria, int, error) {
	r.listPagedHit = true
	r.gotLimit = limit
	r.gotOffset = offset
	r.gotTypePtr = ct
	return r.items, r.total, nil
}

func TestGetCriteriaHandlerPaginates(t *testing.T) {
	repo := &criteriaRepoFake{
		items: []*entity.Criteria{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}},
		total: 137,
	}
	h := NewGetCriteriaHandler(repo)

	items, total, err := h.Handle(context.Background(), GetCriteriaQuery{Limit: 50, Offset: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.listPagedHit {
		t.Fatal("expected ListPaged to be called")
	}
	if repo.gotLimit != 50 || repo.gotOffset != 100 {
		t.Fatalf("limit/offset mismatch: got limit=%d offset=%d", repo.gotLimit, repo.gotOffset)
	}
	if repo.gotTypePtr != nil {
		t.Fatalf("expected nil type filter, got %v", repo.gotTypePtr)
	}
	if total != 137 {
		t.Fatalf("total mismatch: got %d, want 137 (full count, not page size)", total)
	}
	if len(items) != 2 {
		t.Fatalf("items mismatch: got %d, want 2", len(items))
	}
}

func TestGetCriteriaHandlerPassesTypeFilter(t *testing.T) {
	repo := &criteriaRepoFake{}
	h := NewGetCriteriaHandler(repo)

	typeStr := "random"
	_, _, err := h.Handle(context.Background(), GetCriteriaQuery{CriteriaType: &typeStr, Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotTypePtr == nil {
		t.Fatal("expected non-nil type filter passed to repo")
	}
	if repo.gotTypePtr.String() != "random" {
		t.Fatalf("type filter mismatch: got %q, want random", repo.gotTypePtr.String())
	}
}

func TestGetCriteriaHandlerRejectsInvalidType(t *testing.T) {
	repo := &criteriaRepoFake{}
	h := NewGetCriteriaHandler(repo)

	bad := "not-a-real-type"
	_, _, err := h.Handle(context.Background(), GetCriteriaQuery{CriteriaType: &bad, Limit: 50, Offset: 0})
	if err == nil {
		t.Fatal("expected error for invalid criteria type")
	}
	if repo.listPagedHit {
		t.Fatal("ListPaged should not be called when type is invalid")
	}
}
