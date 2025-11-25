package parser

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type GoogleSheetsParser struct {
	service *sheets.Service
}

type Config struct {
	CredentialsJSON []byte
}

func New(cfg Config) (*GoogleSheetsParser, error) {
	ctx := context.Background()

	service, err := sheets.NewService(ctx, option.WithCredentialsJSON(cfg.CredentialsJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	return &GoogleSheetsParser{
		service: service,
	}, nil
}

func (p *GoogleSheetsParser) ParseMenu(ctx context.Context, spreadsheetID, restaurantName string) (*domain.Menu, error) {
	readRange := "A:N" // columns A to N based on given format
	resp, err := p.service.Spreadsheets.Values.Get(spreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read spreadsheet: %w", err)
	}

	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("no data found in spreadsheet")
	}

	menu := &domain.Menu{
		Name:            restaurantName,
		RestaurantID:    generateRestaurantID(restaurantName),
		Products:        []domain.Product{},
		AttributeGroups: []domain.AttributeGroup{},
		Attributes:      []domain.Attribute{},
	}

	var currentProduct *domain.Product
	var currentCategory string
	attributeGroupsMap := make(map[string]*domain.AttributeGroup)
	attributesMap := make(map[string]*domain.Attribute)

	// skip header
	for i := 1; i < len(resp.Values); i++ {
		row := resp.Values[i]
		if len(row) == 0 {
			continue
		}

		// check if this is a category row
		if len(row) == 1 || (len(row) > 1 && row[0] != "" && row[1] == "") {
			currentCategory = fmt.Sprintf("%v", row[0])
			continue
		}

		// check if a product row
		if len(row) > 0 && row[0] != "" && len(row) >= 4 {
			// save prev product if exists
			if currentProduct != nil {
				menu.Products = append(menu.Products, *currentProduct)
			}

			// new product
			product := domain.Product{
				ID:       fmt.Sprintf("%v", row[0]),
				Category: currentCategory,
				Status:   "available",
			}

			if len(row) > 1 {
				product.Name = fmt.Sprintf("%v", row[1])
			}
			if len(row) > 2 {
				isCombo := fmt.Sprintf("%v", row[2])
				product.IsCombo = strings.ToUpper(isCombo) == "TRUE"
			}
			if len(row) > 3 {
				priceStr := fmt.Sprintf("%v", row[3])
				price, err := strconv.ParseFloat(strings.TrimSpace(priceStr), 64)
				if err == nil {
					product.Price = price
				}
			}
			if len(row) > 4 && row[4] != "" {
				product.Description = fmt.Sprintf("%v", row[4])
			}

			currentProduct = &product
			continue
		}

		// check if an attribute group row
		if len(row) > 5 && row[5] != "" {
			attrGroupID := fmt.Sprintf("%v", row[5])

			// check if an attribute group exists
			if _, exists := attributeGroupsMap[attrGroupID]; !exists {
				attrGroup := &domain.AttributeGroup{
					ID:         attrGroupID,
					Attributes: []string{},
				}

				if len(row) > 6 {
					attrGroup.Name = fmt.Sprintf("%v", row[6])
				}
				if len(row) > 7 {
					minStr := fmt.Sprintf("%v", row[7])
					min, err := strconv.Atoi(strings.TrimSpace(minStr))
					if err == nil {
						attrGroup.Min = min
					}
				}
				if len(row) > 8 {
					maxStr := fmt.Sprintf("%v", row[8])
					max, err := strconv.Atoi(strings.TrimSpace(maxStr))
					if err == nil {
						attrGroup.Max = max
					}
				}

				attributeGroupsMap[attrGroupID] = attrGroup
			}

			// attribute
			if len(row) > 9 && row[9] != "" {
				attrID := fmt.Sprintf("%v", row[9])

				if _, exists := attributesMap[attrID]; !exists {
					attr := &domain.Attribute{
						ID: attrID,
					}

					if len(row) > 10 {
						attr.Name = fmt.Sprintf("%v", row[10])
					}
					if len(row) > 11 {
						minStr := fmt.Sprintf("%v", row[11])
						min, err := strconv.Atoi(strings.TrimSpace(minStr))
						if err == nil {
							attr.Min = min
						}
					}
					if len(row) > 12 {
						maxStr := fmt.Sprintf("%v", row[12])
						max, err := strconv.Atoi(strings.TrimSpace(maxStr))
						if err == nil {
							attr.Max = max
						}
					}
					if len(row) > 13 && row[13] != "" {
						priceStr := fmt.Sprintf("%v", row[13])
						price, err := strconv.ParseFloat(strings.TrimSpace(priceStr), 64)
						if err == nil {
							attr.Price = price
						}
					}

					attributesMap[attrID] = attr
				}

				// add attribute to group
				attributeGroupsMap[attrGroupID].Attributes = append(
					attributeGroupsMap[attrGroupID].Attributes,
					attrID,
				)

				// add attribute group to current product if exists
				if currentProduct != nil && !contains(currentProduct.Attributes, attrGroupID) {
					currentProduct.Attributes = append(currentProduct.Attributes, attrGroupID)
				}
			}
		}
	}

	// add last product
	if currentProduct != nil {
		menu.Products = append(menu.Products, *currentProduct)
	}

	for _, attrGroup := range attributeGroupsMap {
		menu.AttributeGroups = append(menu.AttributeGroups, *attrGroup)
	}
	for _, attr := range attributesMap {
		menu.Attributes = append(menu.Attributes, *attr)
	}

	return menu, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateRestaurantID(restaurantName string) string {
	// simple ID generation
	name := strings.ToLower(restaurantName)
	name = strings.ReplaceAll(name, " ", "-")
	return name
}
