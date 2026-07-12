package mall

import "testing"

func TestServiceRecommendedProductsMatchCapturedFirstPage(t *testing.T) {
	service := NewService()
	page := service.SearchPage(SearchRequest{
		CategoryID: "1",
		OrderType:  "default",
		Limit:      PageSize,
	})
	expectedProductIDs := []string{"9011", "9010", "9009", "5024", "5021", "3001", "1000", "27", "26"}
	if len(page.Products) != len(expectedProductIDs) {
		t.Fatalf("expected the captured nine-item recommended page, got %+v", page.Products)
	}
	for index, expectedProductID := range expectedProductIDs {
		product := page.Products[index]
		if product.ProductID != expectedProductID || !product.Recommended || product.CategoryID != "1" || product.Discount != 1 {
			t.Fatalf("expected captured recommended product %s at %d, got %+v", expectedProductID, index, product)
		}
	}
	gem := page.Products[3]
	if gem.Name != "精炼宝石" || gem.Icon != "435.png" || gem.Price != 50 || gem.CouponPrice != 60 {
		t.Fatalf("expected captured 精炼宝石 dual price, got %+v", gem)
	}
	if len(gem.Items) != 1 || gem.Items[0].Name != "精炼宝石" || gem.Items[0].Count != 1 {
		t.Fatalf("expected captured 精炼宝石 item data, got %+v", gem.Items)
	}

	couponOnly := service.SearchPage(SearchRequest{
		CategoryID:      "1",
		DevCurrencyOnly: true,
		OrderType:       "default",
		Limit:           PageSize,
	})
	if len(couponOnly.Products) != 1 || couponOnly.Products[0].ProductID != "5024" {
		t.Fatalf("expected source 点券商品 filter to return 精炼宝石 only, got %+v", couponOnly.Products)
	}
}

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
		t.Fatalf("expected source 点券商品 filter to exclude a 玉币-only product, got %+v", devOnly.Products)
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
