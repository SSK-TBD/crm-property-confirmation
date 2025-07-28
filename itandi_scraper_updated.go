package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

// ITANDIScraperUpdated はITANDI BBの実際の構造に対応したスクレーパー
type ITANDIScraperUpdated struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewITANDIScraperUpdated creates a new updated scraper instance
func NewITANDIScraperUpdated(headless bool) (*ITANDIScraperUpdated, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/Applications/Chromium.app/Contents/MacOS/Chromium"),
		chromedp.Flag("headless", headless),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel2 := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))

	// Create a combined cancel function
	combinedCancel := func() {
		cancel2()
		cancel()
	}

	return &ITANDIScraperUpdated{
		ctx:    ctx,
		cancel: combinedCancel,
	}, nil
}

// Close cleans up resources
func (s *ITANDIScraperUpdated) Close() {
	s.cancel()
}

// NavigateToLogin navigates to the login page
func (s *ITANDIScraperUpdated) NavigateToLogin() error {
	log.Println("Navigating to ITANDI login page...")

	err := chromedp.Run(s.ctx,
		chromedp.Navigate(loginURL),
		chromedp.WaitReady("body"),
	)
	
	if err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}
	
	log.Println("Successfully navigated to login page")
	return nil
}

// ProcessPhoneVerification handles the phone verification process
func (s *ITANDIScraperUpdated) ProcessPhoneVerification(companyName string) error {
	log.Printf("Starting phone verification process for company: %s\n", companyName)
	
	// Wait for company selection dropdown to be available
	err := chromedp.Run(s.ctx,
		chromedp.WaitVisible(`#company_id_select`, chromedp.ByID),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to find company selection dropdown: %w", err)
	}
	
	// Click on company selection dropdown to open it
	err = chromedp.Run(s.ctx,
		chromedp.Click(`#company_id_select + .select2`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to open company dropdown: %w", err)
	}
	
	// Type company name in the search field
	err = chromedp.Run(s.ctx,
		chromedp.SendKeys(`.select2-search__field`, companyName, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second), // Wait for search results
	)
	if err != nil {
		return fmt.Errorf("failed to enter company name: %w", err)
	}
	
	// Select the first result
	err = chromedp.Run(s.ctx,
		chromedp.Click(`.select2-results__option--highlighted`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		// Try alternative selector
		err = chromedp.Run(s.ctx,
			chromedp.Click(`.select2-results__option:first-child`, chromedp.ByQuery),
			chromedp.Sleep(2*time.Second),
		)
		if err != nil {
			return fmt.Errorf("failed to select company: %w", err)
		}
	}
	
	// Wait for store selection to appear and handle it if necessary
	var storeVisible bool
	chromedp.Run(s.ctx,
		chromedp.EvaluateAsDevTools(`document.querySelector('#store_id_select') && !document.querySelector('#store_id_select').closest('.display-none')`, &storeVisible),
	)
	
	if storeVisible {
		log.Println("Store selection is required...")
		err = chromedp.Run(s.ctx,
			chromedp.Click(`#store_id_select + .select2`, chromedp.ByQuery),
			chromedp.Sleep(1*time.Second),
			chromedp.Click(`.select2-results__option:first-child`, chromedp.ByQuery),
			chromedp.Sleep(2*time.Second),
		)
		if err != nil {
			log.Printf("Warning: Failed to select store: %v\n", err)
		}
	}
	
	// Click the "Next" button if it's enabled
	err = chromedp.Run(s.ctx,
		chromedp.WaitVisible(`#btn_login_verify_page`, chromedp.ByID),
		chromedp.Click(`#btn_login_verify_page`, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to click next button: %w", err)
	}
	
	log.Println("Phone verification process initiated")
	return nil
}

// GetPhoneNumber retrieves the phone number for verification
func (s *ITANDIScraperUpdated) GetPhoneNumber() (string, error) {
	log.Println("Getting phone number for verification...")
	
	var phoneNumber string
	err := chromedp.Run(s.ctx,
		chromedp.WaitVisible(`.daihyo-tel-phone`, chromedp.ByQuery),
		chromedp.Text(`.daihyo-tel-phone`, &phoneNumber, chromedp.ByQuery),
	)
	
	if err != nil {
		return "", fmt.Errorf("failed to get phone number: %w", err)
	}
	
	return phoneNumber, nil
}

// SearchPropertyInUpdatedInterface searches for property in the actual ITANDI BB interface
func (s *ITANDIScraperUpdated) SearchPropertyInUpdatedInterface(propertyName string) error {
	log.Printf("Searching for property in ITANDI BB interface: %s\n", propertyName)
	
	// Wait for the main interface to load after phone verification
	time.Sleep(5 * time.Second)
	
	// Look for common search elements in property management systems
	searchSelectors := []string{
		`input[placeholder*="物件"]`,
		`input[placeholder*="検索"]`,
		`input[type="search"]`,
		`input.search`,
		`#search_query`,
		`#property_search`,
		`.search-input input`,
		`[data-search] input`,
	}
	
	var searchFound bool
	for _, selector := range searchSelectors {
		err := chromedp.Run(s.ctx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
		)
		if err == nil {
			log.Printf("Found search input with selector: %s\n", selector)
			
			// Enter search term
			err = chromedp.Run(s.ctx,
				chromedp.SendKeys(selector, propertyName, chromedp.ByQuery),
				chromedp.Sleep(500*time.Millisecond),
				chromedp.KeyEvent("\r"), // Press Enter
			)
			if err == nil {
				searchFound = true
				break
			}
		}
	}
	
	if !searchFound {
		return fmt.Errorf("could not find search input field")
	}
	
	// Wait for search results
	time.Sleep(3 * time.Second)
	
	log.Println("Property search completed")
	return nil
}

// GetUpdatedPropertyDetails extracts property details from the actual ITANDI BB structure
func (s *ITANDIScraperUpdated) GetUpdatedPropertyDetails() (map[string]string, error) {
	log.Println("Getting property details from ITANDI BB interface...")
	
	details := make(map[string]string)
	
	// ITANDI BB specific selectors based on common property management UI patterns
	selectors := map[string][]string{
		"property_name": {
			`.property-title`,
			`.building-name`,
			`.property-name`,
			`h1.title`,
			`h2.title`,
			`.main-title`,
			`[data-property-name]`,
		},
		"address": {
			`.property-address`,
			`.address`,
			`.location`,
			`[data-address]`,
			`.addr`,
		},
		"rent": {
			`.rent`,
			`.price`,
			`.property-price`,
			`.monthly-rent`,
			`[data-rent]`,
			`.price-value`,
		},
		"area": {
			`.area`,
			`.floor-area`,
			`.property-area`,
			`.size`,
			`[data-area]`,
		},
		"layout": {
			`.layout`,
			`.floor-plan`,
			`.property-layout`,
			`.room-type`,
			`[data-layout]`,
		},
		"status": {
			`.status`,
			`.property-status`,
			`.availability`,
			`[data-status]`,
		},
	}
	
	for key, selectorList := range selectors {
		for _, selector := range selectorList {
			var content string
			err := chromedp.Run(s.ctx,
				chromedp.Text(selector, &content, chromedp.ByQuery, chromedp.AtLeast(0)),
			)
			if err == nil && content != "" {
				details[key] = content
				log.Printf("Found %s: %s\n", key, content)
				break
			}
		}
	}
	
	return details, nil
}

// TakeScreenshot takes a screenshot for debugging
func (s *ITANDIScraperUpdated) TakeScreenshot(filename string) error {
	var buf []byte
	
	err := chromedp.Run(s.ctx,
		chromedp.CaptureScreenshot(&buf),
	)
	
	if err != nil {
		return fmt.Errorf("failed to take screenshot: %w", err)
	}
	
	// Save to file
	if err := writeFile(filename, buf); err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}
	
	log.Printf("Screenshot saved to %s\n", filename)
	return nil
}

// GetCurrentURL returns the current page URL
func (s *ITANDIScraperUpdated) GetCurrentURL() (string, error) {
	var url string
	err := chromedp.Run(s.ctx,
		chromedp.Location(&url),
	)
	return url, err
}