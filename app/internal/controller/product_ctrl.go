package controller

import (
	"app/pkg/model"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type ProductController struct {
	products map[string][]*model.Product
	token    string
	mu       sync.RWMutex
}

func NewProductController(token string) *ProductController {
	return &ProductController{
		products: make(map[string][]*model.Product),
		token:    token,
	}
}

func (c *ProductController) GetByName(ctx context.Context, name string) ([]*model.Product, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if product, exists := c.products[name]; exists {
		return product, nil
	}
	offset := 0
	processed := 0
	total := 1

	products := make([]*model.Product, 0)
	for processed < total {
		url := fmt.Sprintf("%s/produtos?offset=%d", OlistERPURL, offset)
		fmt.Printf("URL: %s\n", url)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request to Olist ERP API: %w", err)
		}

		// Add the Bearer token to the request header
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request to Olist ERP API: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("returned non-200 status from Tiny ERP API: %s", resp.Status)
		}
		// Parse the response
		var apiResponse model.ProductResponse

		// Get list of products id from apiResponse

		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			fmt.Printf("Error: %v\n", err)
			return nil, fmt.Errorf("failed to decode Tiny ERP API response: %w", err)
		}
		if len(apiResponse.Itens) == 0 {
			return nil, fmt.Errorf("no product found with name: %s", name)
		}

		// Cache the product
		products = append(products, parseItensFromProductResponse(c, apiResponse, name)...)
		fmt.Printf("Lenght Products: %v\n", len(products))
		offset += len(apiResponse.Itens)
		processed += len(apiResponse.Itens)
		total = apiResponse.Paginacao.Total
	}
	return products, nil
}

func parseItensFromProductResponse(c *ProductController, apiResponse model.ProductResponse, key string) []*model.Product {
	fmt.Println("Reached parseItensFromProductResponse")
	var products []*model.Product
	for _, item := range apiResponse.Itens {
		if !strings.Contains(strings.ToLower(item.NomeProduto), strings.ToLower(key)) {
			continue
		}

		id := item.ID
		url := fmt.Sprintf("%s/produtos/%d", OlistERPURL, id)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}
		defer resp.Body.Close()

		// Parse the response
		var secondaryResponse model.Product
		// TODO REMOVE
		/*
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("failed to read Tiny ERP API response body: %w", err)
				return nil
			}
			fmt.Printf("Response: %s\n", string(bodyBytes))
		*/
		if err := json.NewDecoder(resp.Body).Decode(&secondaryResponse); err != nil {
			continue
		}

		jsonData, _ := json.MarshalIndent(secondaryResponse, "", "  ")
		fmt.Printf("Product: %s\n", jsonData)
		product := &model.Product{
			ID:          secondaryResponse.ID,
			NomeProduto: secondaryResponse.NomeProduto,
			Precos: model.Preco{
				Preco:            secondaryResponse.Precos.Preco,
				PrecoPromocional: secondaryResponse.Precos.PrecoPromocional,
			},
			Dimensoes: model.Dimensoes{
				Largura:           secondaryResponse.Dimensoes.Largura,
				Altura:            secondaryResponse.Dimensoes.Altura,
				Comprimento:       secondaryResponse.Dimensoes.Comprimento,
				Diametro:          secondaryResponse.Dimensoes.Diametro,
				PesoLiquido:       secondaryResponse.Dimensoes.PesoLiquido,
				PesoBruto:         secondaryResponse.Dimensoes.PesoBruto,
				QuantidadeVolumes: secondaryResponse.Dimensoes.QuantidadeVolumes,
			},
			Unidade: secondaryResponse.Unidade,
		}
		products = append(products, product)
	}
	c.products[key] = products
	return products
}
