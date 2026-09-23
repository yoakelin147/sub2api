package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSupplierAccountKinds(t *testing.T) {
	got, err := NormalizeSupplierAccountKinds([]domain.SupplierAccountKind{
		{Platform: " OpenAI ", Type: " APIKEY "},
		{Platform: "openai", Type: "apikey"},
		{Platform: "anthropic", Type: "bedrock"},
		{Platform: "gemini", Type: "service_account"},
	})
	require.NoError(t, err)
	require.Equal(t, []domain.SupplierAccountKind{
		{Platform: "openai", Type: "apikey"},
		{Platform: "anthropic", Type: "bedrock"},
		{Platform: "gemini", Type: "service_account"},
	}, got)
}

func TestSupplierServiceCreateNormalizesInput(t *testing.T) {
	repo := &supplierRepositoryStub{}
	svc := NewSupplierService(repo)

	created, err := svc.Create(context.Background(), CreateSupplierInput{
		Code: " Demo_One ",
		Name: " Demo Supplier ",
		AllowedAccountKinds: []SupplierAccountKind{
			{Platform: " OpenAI ", Type: " APIKEY "},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "demo_one", created.Code)
	require.Equal(t, "Demo Supplier", created.Name)
	require.Equal(t, domain.SupplierStatusActive, created.Status)
	require.Equal(t, []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}, created.AllowedAccountKinds)
}

func TestSupplierServiceCreateGeneratesCode(t *testing.T) {
	svc := NewSupplierService(&supplierRepositoryStub{})
	first, err := svc.Create(context.Background(), CreateSupplierInput{Name: "First"})
	require.NoError(t, err)
	second, err := svc.Create(context.Background(), CreateSupplierInput{Name: "Second"})
	require.NoError(t, err)
	require.Regexp(t, `^supplier-[a-f0-9]{24}$`, first.Code)
	require.Regexp(t, `^supplier-[a-f0-9]{24}$`, second.Code)
	require.NotEqual(t, first.Code, second.Code)
}

func TestParseAccountSupplierFilter(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int64
		valid bool
	}{
		{"", 0, true}, {"  ", 0, true}, {"owned", AccountListSupplierOwned, true},
		{" 42 ", 42, true}, {"0", 0, false}, {"-1", 0, false}, {"invalid", 0, false},
	} {
		got, err := ParseAccountSupplierFilter(tc.input)
		if tc.valid {
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		} else {
			require.Error(t, err)
		}
	}
}

func TestSupplierServiceCreateRejectsInvalidCodeAndStatus(t *testing.T) {
	svc := NewSupplierService(&supplierRepositoryStub{})

	_, err := svc.Create(context.Background(), CreateSupplierInput{Code: "not valid!", Name: "Supplier"})
	require.ErrorIs(t, err, ErrSupplierCodeInvalid)

	_, err = svc.Create(context.Background(), CreateSupplierInput{
		Code: "valid", Name: "Supplier", Status: "deleted",
	})
	require.ErrorIs(t, err, ErrSupplierStatusInvalid)
}

type supplierRepositoryStub struct {
	created   *Supplier
	listItems []Supplier
	stats     map[int64]SupplierStats
}

func (r *supplierRepositoryStub) Create(_ context.Context, supplier *Supplier) error {
	supplier.ID = 1
	r.created = supplier
	return nil
}

func (r *supplierRepositoryStub) GetByID(context.Context, int64) (*Supplier, error) {
	return nil, ErrSupplierNotFound
}

func (r *supplierRepositoryStub) GetByTokenSelector(context.Context, string) (*Supplier, error) {
	return nil, ErrSupplierNotFound
}

func (r *supplierRepositoryStub) Update(context.Context, *Supplier) error { return nil }
func (r *supplierRepositoryStub) Delete(context.Context, int64) error     { return nil }

func (r *supplierRepositoryStub) List(
	context.Context,
	pagination.PaginationParams,
	SupplierListFilters,
) ([]Supplier, *pagination.PaginationResult, error) {
	return r.listItems, &pagination.PaginationResult{Total: int64(len(r.listItems)), Page: 1, PageSize: 20}, nil
}

func (r *supplierRepositoryStub) GetStatsBySupplierIDs(context.Context, []int64) (map[int64]SupplierStats, error) {
	return r.stats, nil
}

func (r *supplierRepositoryStub) UpdateTokenLastUsed(context.Context, int64, time.Time) error {
	return nil
}

func TestNormalizeSupplierAccountKindsRejectsInternalAndUnknownKinds(t *testing.T) {
	tests := []domain.SupplierAccountKind{
		{Platform: PlatformComposite, Type: AccountTypeAPIKey},
		{Platform: PlatformOpenAI, Type: AccountTypeBedrock},
		{Platform: "unknown", Type: AccountTypeAPIKey},
		{Platform: PlatformOpenAI, Type: "cookie"},
	}

	for _, kind := range tests {
		_, err := NormalizeSupplierAccountKinds([]domain.SupplierAccountKind{kind})
		require.ErrorIs(t, err, ErrSupplierAccountKindInvalid)
	}
}

func TestSupplierServiceListAttachesOperationalStats(t *testing.T) {
	want := SupplierStats{MemberCount: 2, AccountCount: 8, PendingCount: 3, SchedulableCount: 4, ErrorCount: 1}
	repo := &supplierRepositoryStub{
		listItems: []Supplier{{ID: 7, Code: "vendor"}},
		stats:     map[int64]SupplierStats{7: want},
	}

	items, _, err := NewSupplierService(repo).List(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, SupplierListFilters{})
	require.NoError(t, err)
	require.Equal(t, want, items[0].Stats)
}
