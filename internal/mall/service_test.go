package mall

import "testing"

func TestServiceSearchCapturedFashionProducts(t *testing.T) {
	service := NewService()

	count := service.SearchCount(SearchRequest{
		CategoryID: "all",
		Keyword:    "超时空",
	})
	if count.Count != 1 {
		t.Fatalf("expected one captured fashion product count, got %+v", count)
	}

	page := service.SearchPage(SearchRequest{
		CategoryID: "all",
		Keyword:    "超时空",
		Limit:      99,
	})
	if page.Limit != PageSize || len(page.Products) != 1 {
		t.Fatalf("expected one captured fashion search result with source page size clamp, got %+v", page)
	}
	product := page.Products[0]
	if product.ProductID != "8019" || product.Name != "超时空要塞" || product.Icon != "1205.png" || product.Price != 150 || product.Currency != SourceYubiCurrencyName {
		t.Fatalf("expected captured 超时空要塞 product fields, got %+v", product)
	}
	if product.Description == "" || product.Description[:len("f_i_超时空要塞")] != "f_i_超时空要塞" {
		t.Fatalf("expected captured source item description, got %q", product.Description)
	}
	if len(product.Items) != 1 || product.Items[0].Name != "超时空要塞" || product.Items[0].Display != "1205.png" || product.Items[0].Count != 1 {
		t.Fatalf("expected captured product source item details, got %+v", product.Items)
	}

	devOnly := service.SearchPage(SearchRequest{
		CategoryID:      "all",
		Keyword:         "超时空",
		DevCurrencyOnly: true,
	})
	if len(devOnly.Products) != 0 {
		t.Fatalf("expected captured 玉币 products to be excluded from dev-currency-only search, got %+v", devOnly.Products)
	}

	summer, ok := service.FindProduct("7033")
	if !ok {
		t.Fatal("expected captured 盛夏缤纷 product to be findable by source product id")
	}
	if summer.Name != "盛夏缤纷" || summer.Icon != "729.png" || summer.Price != 100 || summer.Currency != SourceYubiCurrencyName {
		t.Fatalf("expected captured 盛夏缤纷 product fields, got %+v", summer)
	}
	if len(summer.Items) != 1 || summer.Items[0].Name != "盛夏缤纷" || summer.Items[0].Display != "729.png" || summer.Items[0].Count != 1 {
		t.Fatalf("expected captured summer product source item details, got %+v", summer.Items)
	}
}
