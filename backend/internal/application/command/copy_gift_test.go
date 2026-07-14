package command

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

func TestCopyGiftHandlerRejectsInvalidCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  CopyGiftCommand
		want error
	}{
		{name: "zero gift ID", cmd: CopyGiftCommand{CopiesCount: 1}, want: ErrInvalidGiftCopySourceID},
		{name: "zero copies", cmd: CopyGiftCommand{GiftID: 1}, want: ErrInvalidGiftCopiesCount},
		{name: "too many copies", cmd: CopyGiftCommand{GiftID: 1, CopiesCount: 101}, want: ErrInvalidGiftCopiesCount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &giftCopyRepositoryFake{}
			handler := NewCopyGiftHandler(repo)

			_, err := handler.Handle(context.Background(), tt.cmd)

			if !errors.Is(err, tt.want) {
				t.Fatalf("Handle() error = %v, want %v", err, tt.want)
			}
			if repo.calls != 0 {
				t.Fatalf("Copy() calls = %d, want 0", repo.calls)
			}
		})
	}
}

func TestCopyGiftHandlerMapsExpectedRepositoryErrors(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		want    error
	}{
		{name: "source gift missing", repoErr: repository.ErrGiftNotFound, want: ErrGiftNotFound},
		{name: "source has place constraint", repoErr: repository.ErrGiftCopyHasPlaceConstraint, want: ErrGiftCopyHasPlaceConstraint},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &giftCopyRepositoryFake{err: tt.repoErr}
			handler := NewCopyGiftHandler(repo)

			_, err := handler.Handle(context.Background(), CopyGiftCommand{GiftID: 7, CopiesCount: 2})

			if !errors.Is(err, tt.want) {
				t.Fatalf("Handle() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCopyGiftHandlerReturnsCopyMetadata(t *testing.T) {
	repo := &giftCopyRepositoryFake{
		result: repository.GiftCopyResult{
			EventID:      77,
			ReviewStatus: entity.GiftReviewStatusApproved,
		},
	}
	handler := NewCopyGiftHandler(repo)

	result, err := handler.Handle(context.Background(), CopyGiftCommand{GiftID: 7, CopiesCount: 3})

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.EventID != 77 || result.ReviewStatus != entity.GiftReviewStatusApproved || result.CreatedCount != 3 {
		t.Fatalf("Handle() result = %#v, want event 77, approved, 3 copies", result)
	}
	if repo.sourceGiftID != 7 || repo.copiesCount != 3 {
		t.Fatalf("Copy() arguments = gift %d, copies %d, want gift 7, copies 3", repo.sourceGiftID, repo.copiesCount)
	}
}

type giftCopyRepositoryFake struct {
	result       repository.GiftCopyResult
	err          error
	calls        int
	sourceGiftID uint
	copiesCount  int
}

func (r *giftCopyRepositoryFake) Copy(ctx context.Context, sourceGiftID uint, copiesCount int) (repository.GiftCopyResult, error) {
	r.calls++
	r.sourceGiftID = sourceGiftID
	r.copiesCount = copiesCount
	return r.result, r.err
}
