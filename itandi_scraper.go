package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	loginURL      = "https://itandi-accounts.com/"
	loginEmail    = "info@xxx"
	loginPassword = "xxx"
)

// ITANDIScraper はITANDI BBのスクレーパー
type ITANDIScraper struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewITANDIScraper creates a new scraper instance
func NewITANDIScraper(headless bool) (*ITANDIScraper, error) {
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

	return &ITANDIScraper{
		ctx:    ctx,
		cancel: combinedCancel,
	}, nil
}

// Close cleans up resources
func (s *ITANDIScraper) Close() {
	s.cancel()
}

// NavigateToLogin navigates to the login page
func (s *ITANDIScraper) NavigateToLogin() error {
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

// TakeScreenshot takes a screenshot for debugging
func (s *ITANDIScraper) TakeScreenshot(filename string) error {
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

// Login performs flexible login to ITANDI BB
func (s *ITANDIScraper) Login() error {
	log.Println("Starting adaptive login process...")

	// First, determine what type of login interface is available
	time.Sleep(2 * time.Second)

	// Check for email/password inputs
	var hasEmailInput, hasPasswordInput bool
	chromedp.Run(s.ctx,
		chromedp.EvaluateAsDevTools(`
			document.querySelector('input[type="email"], input[name="email"], input[id*="email"], input[placeholder*="email"], input[placeholder*="メール"]') !== null
		`, &hasEmailInput),
	)

	chromedp.Run(s.ctx,
		chromedp.EvaluateAsDevTools(`
			document.querySelector('input[type="password"], input[name="password"], input[id*="password"], input[placeholder*="password"], input[placeholder*="パスワード"]') !== null
		`, &hasPasswordInput),
	)

	if hasEmailInput && hasPasswordInput {
		log.Println("Found email/password login form - attempting email login")
		return s.performEmailPasswordLogin()
	}

	// Check for company selection (phone verification system)
	var hasCompanySelect bool
	chromedp.Run(s.ctx,
		chromedp.EvaluateAsDevTools(`
			document.querySelector('#company_id_select, select[name="company_id"]') !== null
		`, &hasCompanySelect),
	)

	if hasCompanySelect {
		log.Println("Found company selection form - this appears to be phone verification system")
		log.Println("Please use -updated flag for phone verification system")
		return fmt.Errorf("phone verification system detected - use -updated flag")
	}

	// Look for any clickable login elements
	log.Println("Looking for login buttons or links...")

	loginElements := []string{
		`a[href*="login"]`,
		`a[href*="sign_in"]`,
		`button:contains("ログイン")`,
		`button:contains("Login")`,
		`button:contains("Sign In")`,
		`.login-btn`,
		`.signin-btn`,
	}

	for _, selector := range loginElements {
		err := chromedp.Run(s.ctx,
			chromedp.Click(selector, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Clicked login element: %s\n", selector)
			time.Sleep(3 * time.Second)

			// Check again for email/password inputs after clicking
			chromedp.Run(s.ctx,
				chromedp.EvaluateAsDevTools(`
					document.querySelector('input[type="email"], input[name="email"], input[id*="email"]') !== null
				`, &hasEmailInput),
			)
			chromedp.Run(s.ctx,
				chromedp.EvaluateAsDevTools(`
					document.querySelector('input[type="password"], input[name="password"], input[id*="password"]') !== null
				`, &hasPasswordInput),
			)

			if hasEmailInput && hasPasswordInput {
				log.Println("Email/password form appeared after clicking - attempting login")
				return s.performEmailPasswordLogin()
			}
		}
	}

	return fmt.Errorf("no compatible login method found - ITANDI BB may require phone verification")
}

// performEmailPasswordLogin handles the actual email/password login
func (s *ITANDIScraper) performEmailPasswordLogin() error {
	log.Println("Performing email/password login...")

	// Find and fill email
	emailSelectors := []string{
		`input[type="email"]`,
		`input[name="email"]`,
		`input[id*="email"]`,
		`input[placeholder*="email"]`,
		`input[placeholder*="メール"]`,
	}

	var emailFilled bool
	for _, selector := range emailSelectors {
		err := chromedp.Run(s.ctx,
			chromedp.SendKeys(selector, loginEmail, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Email entered using: %s\n", selector)
			emailFilled = true
			break
		}
	}

	if !emailFilled {
		return fmt.Errorf("could not fill email field")
	}

	// Find and fill password
	passwordSelectors := []string{
		`input[type="password"]`,
		`input[name="password"]`,
		`input[id*="password"]`,
		`input[placeholder*="password"]`,
		`input[placeholder*="パスワード"]`,
	}

	var passwordFilled bool
	for _, selector := range passwordSelectors {
		err := chromedp.Run(s.ctx,
			chromedp.SendKeys(selector, loginPassword, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Password entered using: %s\n", selector)
			passwordFilled = true
			break
		}
	}

	if !passwordFilled {
		return fmt.Errorf("could not fill password field")
	}

	time.Sleep(1 * time.Second)

	// Submit form
	submitSelectors := []string{
		`button[type="submit"]`,
		`input[type="submit"]`,
		`button:contains("ログイン")`,
		`button:contains("Login")`,
		`button:contains("Sign In")`,
		`.login-btn`,
		`.submit-btn`,
	}

	var submitted bool
	for _, selector := range submitSelectors {
		err := chromedp.Run(s.ctx,
			chromedp.Click(selector, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Submitted using: %s\n", selector)
			submitted = true
			break
		}
	}

	if !submitted {
		// Try Enter key
		err := chromedp.Run(s.ctx,
			chromedp.KeyEvent("\r"),
		)
		if err == nil {
			log.Println("Submitted using Enter key")
			submitted = true
		}
	}

	if !submitted {
		return fmt.Errorf("could not submit login form")
	}

	// Wait for response
	time.Sleep(5 * time.Second)

	log.Println("Email/password login completed")
	return nil
}

// SearchProperty searches for a property by name following ITANDI BB's actual flow
func (s *ITANDIScraper) SearchProperty(propertyName string) error {
	log.Printf("Searching for property: %s\n", propertyName)

	var err error

	// Step 1: Wait for page to stabilize and ensure we're on the correct page
	time.Sleep(2 * time.Second)

	// Check if we're on the top page
	url, _ := s.GetPageURL()
	log.Printf("Current URL: %s\n", url)

	// If we're not on the top page, navigate to it
	if !strings.Contains(url, "/top") {
		log.Println("Navigating to ITANDI BB top page...")
		err := chromedp.Run(s.ctx,
			chromedp.Navigate("https://itandibb.com/top"),
			chromedp.WaitReady("body"),
		)
		if err != nil {
			return fmt.Errorf("failed to navigate to top page: %w", err)
		}
		time.Sleep(2 * time.Second)
	}

	// Step 2: Find and click the rental module's list search button
	log.Println("Looking for rental module list search button...")

	// Try various possible selectors for the list search button
	listSearchSelectors := []string{
		`a:contains("リスト検索")`,
		`button:contains("リスト検索")`,
		`.rental-module a:contains("リスト検索")`,
		`[class*="rental"] a:contains("検索")`,
		`a[href*="list"], a[href*="search"]`,
		`div:contains("賃貸") a:contains("検索")`,
		// More specific selectors
		`a[href*="/properties"], a[href*="/search"]`,
		`.module-rental a`,
		`#rental-search`,
	}

	var clicked bool
	for _, selector := range listSearchSelectors {
		err := chromedp.Run(s.ctx,
			chromedp.Click(selector, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Clicked list search using selector: %s\n", selector)
			clicked = true
			break
		}
	}

	if !clicked {
		// Try JavaScript approach
		err := chromedp.Run(s.ctx,
			chromedp.Evaluate(`
				const links = Array.from(document.querySelectorAll('a'));
				const listSearchLink = links.find(a => 
					a.textContent.includes('リスト検索') || 
					a.textContent.includes('検索') && a.closest('[class*="rental"], [class*="賃貸"]')
				);
				if (listSearchLink) {
					listSearchLink.click();
					true;
				} else {
					false;
				}
			`, &clicked),
		)
		if err != nil || !clicked {
			return fmt.Errorf("could not find list search button in rental module")
		}
		log.Println("Clicked list search using JavaScript")
	}

	// Wait for navigation to search page
	time.Sleep(3 * time.Second)

	// Step 3: FIRST - Close modal advertisements on the list page
	log.Println("=== Step 3-1: Closing modal advertisements on list page ===")

	// Use a more direct approach to close the specific ITANDI modal
	var modalClosed bool
	for attempt := 1; attempt <= 5; attempt++ {
		log.Printf("Modal closing attempt %d/5...\n", attempt)

		// Check if modal is still visible
		var hasVisibleModal bool
		err = chromedp.Run(s.ctx,
			chromedp.Evaluate(`
				(() => {
					// Look for the specific ITANDI modal text
					const modalElements = document.querySelectorAll('*');
					for (let el of modalElements) {
						if (el.textContent && el.textContent.includes('イタンジ売却査定')) {
							const style = window.getComputedStyle(el);
							if (style.display !== 'none' && style.visibility !== 'hidden') {
								return true;
							}
						}
					}
					return false;
				})()
			`, &hasVisibleModal),
		)

		if err == nil && !hasVisibleModal {
			log.Println("Modal successfully closed!")
			modalClosed = true
			break
		}

		// Try multiple strategies to close the modal
		var closeSuccess bool

		// Strategy 1: Direct selector approach
		closeSelectors := []string{
			`button[aria-label="close"]`,
			`button[title="close"]`,
			`span:contains("×")`,
			`div:contains("×")`,
			`[class*="close"]`,
			`.modal-close`,
			`[role="dialog"] button`,
		}

		for _, selector := range closeSelectors {
			err = chromedp.Run(s.ctx,
				chromedp.Click(selector, chromedp.ByQuery, chromedp.AtLeast(0)),
			)
			if err == nil {
				log.Printf("Closed modal using selector: %s\n", selector)
				closeSuccess = true
				break
			}
		}

		// Strategy 2: JavaScript approach to find the exact close button
		if !closeSuccess {
			err = chromedp.Run(s.ctx,
				chromedp.Evaluate(`
					(() => {
						// Look for the modal with the specific text
						const allElements = Array.from(document.querySelectorAll('*'));
						
						// Find all elements with the modal text
						const modalElements = allElements.filter(el => 
							el.textContent && el.textContent.includes('イタンジ売却査定') && 
							el.textContent.includes('信頼される査定書を')
						);
						
						if (modalElements.length === 0) return false;
						
						// Find the modal container
						let modalContainer = null;
						for (let el of modalElements) {
							let current = el;
							while (current && current.parentElement) {
								const style = window.getComputedStyle(current.parentElement);
								if (style.position === 'fixed' && 
									current.parentElement.offsetWidth > 300 && 
									current.parentElement.offsetHeight > 200) {
									modalContainer = current.parentElement;
									break;
								}
								current = current.parentElement;
							}
							if (modalContainer) break;
						}
						
						if (!modalContainer) return false;
						
						// Look for close elements - be very broad
						const allElementsInModal = Array.from(modalContainer.querySelectorAll('*'));
						
						for (let el of allElementsInModal) {
							const rect = el.getBoundingClientRect();
							const modalRect = modalContainer.getBoundingClientRect();
							
							// Check if element is in top-right corner of modal
							if (rect.right >= modalRect.right - 40 &&
								rect.top <= modalRect.top + 40 &&
								rect.width >= 15 && rect.width <= 50 &&
								rect.height >= 15 && rect.height <= 50) {
								
								try {
									el.click();
									return true;
								} catch (e) {
									// Try parent or children
									try {
										if (el.parentElement) {
											el.parentElement.click();
											return true;
										}
									} catch (e2) {
										// Continue
									}
								}
							}
						}
						
						return false;
					})()
				`, &closeSuccess),
			)
		}

		if err == nil && closeSuccess {
			log.Printf("Clicked close button on attempt %d\n", attempt)
		} else {
			log.Printf("Failed to find/click close button on attempt %d\n", attempt)
		}

		time.Sleep(2 * time.Second)
	}

	if !modalClosed {
		log.Println("Warning: Modal may still be visible, but continuing with search...")
	}

	// Take screenshot after modal closing
	if err := s.TakeScreenshot("after_modal_close.png"); err != nil {
		log.Println("Warning: Failed to take screenshot after modal close:", err)
	}

	// Step 3-2: Find and fill the property name search field
	log.Println("=== Step 3-2: Entering property name ===")

	// Try various selectors for property name input
	propertyNameSelectors := []string{
		`input[placeholder*="物件名"]`,
		`input[placeholder*="カナ検索"]`,
		`input[name*="property"]`,
		`input[name*="building"]`,
		`input[type="text"][placeholder*="物件"]`,
		`input[type="text"]:has(~ label:contains("物件名"))`,
		// More generic selectors
		`form input[type="text"]:first`,
		`.search-form input[type="text"]`,
		`#property_name, #building_name`,
	}

	var inputFilled bool
	for _, selector := range propertyNameSelectors {
		err := chromedp.Run(s.ctx,
			chromedp.SendKeys(selector, propertyName, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Entered property name using selector: %s\n", selector)
			inputFilled = true
			break
		}
	}

	if !inputFilled {
		// Try to find input by looking for labels
		err = chromedp.Run(s.ctx,
			chromedp.Evaluate(`
				const labels = Array.from(document.querySelectorAll('label'));
				const propertyLabel = labels.find(label => 
					label.textContent.includes('物件名') || 
					label.textContent.includes('カナ検索')
				);
				if (propertyLabel) {
					const input = propertyLabel.parentElement.querySelector('input[type="text"]') ||
								  document.getElementById(propertyLabel.getAttribute('for'));
					if (input) {
						input.value = '${propertyName}';
						input.dispatchEvent(new Event('input', { bubbles: true }));
						input.dispatchEvent(new Event('change', { bubbles: true }));
						true;
					}
				}
				false;
			`, nil),
		)
		if err != nil {
			return fmt.Errorf("could not find property name input field")
		}
		log.Println("Entered property name using JavaScript")
	}

	// Take screenshot after input
	if err := s.TakeScreenshot("after_property_input.png"); err != nil {
		log.Println("Warning: Failed to take screenshot after input:", err)
	}

	time.Sleep(1 * time.Second)

	// Step 3-3: Click the search button (避开条件保存按钮)
	log.Println("=== Step 3-3: Clicking search button ===")

	// First try specific selectors for the orange search button (avoiding 条件保存)
	specificSearchSelectors := []string{
		`button[style*="background-color: rgb(255, 145, 65)"]`, // Orange background
		`button[style*="background: rgb(255, 145, 65)"]`,
		`button.MuiButton-containedPrimary:contains("検索")`, // Material-UI primary button
		`button[class*="orange"]:contains("検索")`,
		`button[class*="primary"]:contains("検索"):not(:contains("削除")):not(:contains("保存"))`,
		`input[type="submit"][value="検索"][style*="background"]`,
		`button:contains("検索"):not(:contains("削除")):not(:contains("保存")):not(:contains("条件"))`,
	}

	var searchClicked bool
	for _, selector := range specificSearchSelectors {
		err := chromedp.Run(s.ctx,
			chromedp.Click(selector, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Clicked search button using specific selector: %s\n", selector)
			searchClicked = true
			break
		}
	}

	// If specific selectors fail, use JavaScript to find the correct search button
	if !searchClicked {
		log.Println("Trying JavaScript approach to find search button...")
		var jsSearchSuccess bool
		err = chromedp.Run(s.ctx,
			chromedp.Evaluate(`
				(() => {
					// Look specifically for the orange search button (avoid 条件保存)
					const allButtons = document.querySelectorAll('button, input[type="submit"], a');
					
					// Strategy 1: Prioritize orange/colored search button
					for (let btn of allButtons) {
						const text = btn.textContent ? btn.textContent.trim() : '';
						const value = btn.value ? btn.value.trim() : '';
						const style = window.getComputedStyle(btn);
						const rect = btn.getBoundingClientRect();
						const bgColor = style.backgroundColor;
						
						// Very strict criteria: must contain "検索" and NOT contain prohibited words
						if ((text.includes('検索') || value.includes('検索')) && 
							!text.includes('条件') && !text.includes('削除') && !text.includes('全削除') && 
							!text.includes('保存') && !text.includes('クリア')) {
							
							// High priority: orange background color (exact match)
							if (rect.width > 0 && rect.height > 0 && 
								(bgColor === 'rgb(255, 145, 65)' || // Exact orange color
								 bgColor.includes('rgb(255, 145, 65)') ||
								 style.backgroundColor === 'rgb(255, 145, 65)')) {
								
								console.log('Found orange search button:', text, bgColor);
								try {
									btn.click();
									return true;
								} catch (e) {
									console.log('Error clicking orange button:', e);
								}
							}
						}
					}
					
					// Strategy 1.5: Look for primary/contained buttons with 検索
					for (let btn of allButtons) {
						const text = btn.textContent ? btn.textContent.trim() : '';
						const value = btn.value ? btn.value.trim() : '';
						const rect = btn.getBoundingClientRect();
						
						if ((text.includes('検索') || value.includes('検索')) && 
							!text.includes('条件') && !text.includes('削除') && !text.includes('保存') &&
							rect.width > 0 && rect.height > 0) {
							
							// Look for Material-UI or styled buttons
							if (btn.className.includes('primary') || btn.className.includes('contained') ||
								btn.className.includes('MuiButton') || btn.className.includes('orange')) {
								
								console.log('Found primary/contained search button:', text, btn.className);
								try {
									btn.click();
									return true;
								} catch (e) {
									console.log('Error clicking primary button:', e);
								}
							}
						}
					}
					
					// Strategy 2: Look for the button that's visually the main search action
					for (let btn of allButtons) {
						const text = btn.textContent ? btn.textContent.trim() : '';
						const rect = btn.getBoundingClientRect();
						const style = window.getComputedStyle(btn);
						
						// Look for button at bottom-right with "検索" text
						if (text.includes('検索') && 
							rect.bottom > window.innerHeight * 0.6 && 
							rect.right > window.innerWidth * 0.5 &&
							!text.includes('削除') && !text.includes('保存') && !text.includes('条件')) {
							
							try {
								btn.click();
								return true;
							} catch (e) {
								// Continue
							}
						}
					}
					
					// Strategy 3: Find button with search icon (SVG or specific styling)
					for (let btn of allButtons) {
						if (btn.querySelector('svg') || btn.innerHTML.includes('search') || 
							btn.getAttribute('aria-label') === 'search') {
							const text = btn.textContent ? btn.textContent.trim() : '';
							if (text.includes('検索') || text === '') {
								try {
									btn.click();
									return true;
								} catch (e) {
									// Continue
								}
							}
						}
					}
					
					return false;
				})()
			`, &jsSearchSuccess),
		)

		if err == nil && jsSearchSuccess {
			log.Println("Successfully clicked search button using JavaScript")
			searchClicked = true
		}
	}

	if !searchClicked {
		// Try pressing Enter
		err := chromedp.Run(s.ctx,
			chromedp.KeyEvent("\r"),
		)
		if err == nil {
			log.Println("Submitted search using Enter key")
			searchClicked = true
		}
	}

	if !searchClicked {
		return fmt.Errorf("could not submit search")
	}

	// Wait for search results
	time.Sleep(5 * time.Second)

	log.Println("Property search completed")
	return nil
}

// GetPropertyDOM retrieves specific DOM elements from property details
func (s *ITANDIScraper) GetPropertyDOM(selector string) (string, error) {
	log.Printf("Getting DOM element with selector: %s\n", selector)

	var content string
	err := chromedp.Run(s.ctx,
		chromedp.Text(selector, &content, chromedp.ByQuery),
	)
	if err != nil {
		return "", fmt.Errorf("failed to get DOM content: %w", err)
	}

	return content, nil
}

// GetPropertyDetails extracts multiple DOM elements from ITANDI BB search results
func (s *ITANDIScraper) GetPropertyDetails() (map[string]string, error) {
	log.Println("Getting property details from search results...")

	details := make(map[string]string)

	// Wait for search results to load
	time.Sleep(3 * time.Second)

	// Aggressively close all modals that might be blocking the results
	log.Println("Aggressively closing all modals before extracting results...")
	for i := 0; i < 3; i++ {
		s.closeModalAdsQuick()
		time.Sleep(500 * time.Millisecond)
	}

	// Get current URL to understand which page we're on
	url, _ := s.GetPageURL()
	details["current_page_url"] = url

	// First, check if there are any search results at all
	log.Println("Checking for search results...")
	var resultsData interface{}

	err := chromedp.Run(s.ctx,
		chromedp.Evaluate(`
			(() => {
				// Check for "no results" message
				const noResultsSelectors = [
					'*:contains("検索結果がありませんでした")',
					'*:contains("該当する物件がありません")',
					'*:contains("見つかりませんでした")',
					'*:contains("0件")',
					'.no-results',
					'.empty-results'
				];
				
				let noResultsFound = false;
				let message = '';
				
				// Check for no results message
				const allElements = document.querySelectorAll('*');
				for (let el of allElements) {
					const text = el.textContent;
					if (text && (text.includes('検索結果がありませんでした') || 
								text.includes('該当する物件がありません') ||
								text.includes('見つかりませんでした') ||
								text.includes('ご希望の条件に一致する検索結果がありませんでした'))) {
						noResultsFound = true;
						message = text.trim();
						break;
					}
				}
				
				// Also check for actual property listing table
				const resultTables = document.querySelectorAll('table');
				let hasPropertyTable = false;
				for (let table of resultTables) {
					// Look for table headers that suggest property listings
					const headers = table.querySelectorAll('th');
					for (let header of headers) {
						const headerText = header.textContent;
						if (headerText && (headerText.includes('物件') || headerText.includes('賃料') || 
										  headerText.includes('間取') || headerText.includes('面積'))) {
							hasPropertyTable = true;
							break;
						}
					}
					if (hasPropertyTable) break;
				}
				
				return {
					hasResults: hasPropertyTable && !noResultsFound,
					noResultsMessage: message,
					tableCount: resultTables.length
				};
			})()
		`, &resultsData),
	)

	if err != nil {
		log.Printf("Error checking for results: %v\n", err)
	} else {
		// Parse the JavaScript result
		if resultMap, ok := resultsData.(map[string]interface{}); ok {
			if hasRes, ok := resultMap["hasResults"].(bool); ok && hasRes {
				log.Println("Search results found - proceeding with extraction")
				details["search_status"] = "Results found"
			} else {
				log.Println("No search results found")
				details["search_status"] = "No results found"
				if msg, ok := resultMap["noResultsMessage"].(string); ok && msg != "" {
					details["no_results_message"] = msg
				}
			}
			if tableCount, ok := resultMap["tableCount"].(float64); ok {
				details["table_count"] = fmt.Sprintf("%.0f", tableCount)
			}
		}
	}

	// If no results found, return early
	if details["search_status"] == "No results found" {
		log.Println("No search results found - returning early")

		// Get page title for context
		var pageTitle string
		chromedp.Run(s.ctx,
			chromedp.Title(&pageTitle),
		)
		if pageTitle != "" {
			details["page_title"] = pageTitle
		}

		return details, nil
	}

	// ITANDI BB specific selectors for search results (only if results exist)
	selectors := map[string][]string{
		"search_count": {
			`.search-count`,
			`.result-count`,
			`[class*="count"]`,
			`span:contains("件")`,
		},
		"property_name": {
			`.property-name`,
			`.building-name`,
			`[class*="building"]`,
			`td[data-label*="物件名"]`,
			`.list-table td:nth-child(2)`,
			`a[href*="/rent_rooms/"]`,
		},
		"address": {
			`.property-address`,
			`.address`,
			`[class*="address"]`,
			`td[data-label*="住所"]`,
			`.list-table td:nth-child(3)`,
		},
		"rent": {
			`.rent`,
			`.price`,
			`[class*="rent"]`,
			`td[data-label*="賃料"]`,
			`.list-table td:nth-child(4)`,
		},
		"layout": {
			`.layout`,
			`.floor-plan`,
			`[class*="layout"]`,
			`td[data-label*="間取"]`,
			`.list-table td:nth-child(5)`,
		},
		"area": {
			`.area`,
			`.floor-area`,
			`[class*="area"]`,
			`td[data-label*="面積"]`,
			`.list-table td:nth-child(6)`,
		},
		"station": {
			`.station`,
			`[class*="station"]`,
			`td[data-label*="最寄駅"]`,
			`.list-table td:nth-child(7)`,
		},
		"management_company": {
			`.management-company`,
			`[class*="management"]`,
			`td[data-label*="管理会社"]`,
			`.list-table td:nth-child(8)`,
		},
	}

	// Try to get each piece of information
	for key, selectorList := range selectors {
		for _, selector := range selectorList {
			var content string
			err := chromedp.Run(s.ctx,
				chromedp.Text(selector, &content, chromedp.ByQuery, chromedp.AtLeast(0)),
			)
			if err == nil && content != "" && content != " " {
				details[key] = strings.TrimSpace(content)
				log.Printf("Found %s: %s (using selector: %s)\n", key, content, selector)
				break
			}
		}
	}

	// Try to get all property names if multiple results
	var propertyNames []string
	err = chromedp.Run(s.ctx,
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a[href*="/rent_rooms/"], .property-name, [class*="building"]'))
				.map(el => el.textContent.trim())
				.filter(text => text.length > 0)
		`, &propertyNames),
	)

	if err == nil && len(propertyNames) > 0 {
		details["all_properties"] = strings.Join(propertyNames, ", ")
		details["property_count"] = fmt.Sprintf("%d", len(propertyNames))
		log.Printf("Found %d properties in search results\n", len(propertyNames))
	}

	// Try to get search result summary
	var resultSummary string
	chromedp.Run(s.ctx,
		chromedp.Text(`body`, &resultSummary, chromedp.ByQuery),
	)

	if strings.Contains(resultSummary, "0件") || strings.Contains(resultSummary, "該当する物件がありません") {
		details["search_status"] = "No results found"
		log.Println("No search results found")
	} else if strings.Contains(resultSummary, "件") {
		details["search_status"] = "Results found"
	}

	// Get page title
	var pageTitle string
	chromedp.Run(s.ctx,
		chromedp.Title(&pageTitle),
	)
	if pageTitle != "" {
		details["page_title"] = pageTitle
	}

	return details, nil
}

// AnalyzePageStructure analyzes the current page structure and logs available elements
func (s *ITANDIScraper) AnalyzePageStructure() error {
	log.Println("Analyzing page structure...")

	// Get page HTML for analysis
	var html string
	err := chromedp.Run(s.ctx,
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("failed to get page HTML: %w", err)
	}

	// Save HTML to file for detailed analysis
	if err := writeFile("page_structure.html", []byte(html)); err != nil {
		log.Printf("Warning: Failed to save HTML: %v\n", err)
	} else {
		log.Println("Page HTML saved to page_structure.html")
	}

	// Try to find common form elements
	var inputs []string
	err = chromedp.Run(s.ctx,
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('input')).map(input => ({
				type: input.type,
				name: input.name,
				id: input.id,
				placeholder: input.placeholder,
				className: input.className
			}))
		`, &inputs),
	)
	if err == nil {
		log.Printf("Found %d input elements\n", len(inputs))
	}

	// Try to find buttons
	var buttons []string
	err = chromedp.Run(s.ctx,
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('button, input[type="submit"]')).map(btn => ({
				text: btn.textContent || btn.value,
				type: btn.type,
				className: btn.className,
				id: btn.id
			}))
		`, &buttons),
	)
	if err == nil {
		log.Printf("Found %d button elements\n", len(buttons))
	}

	return nil
}

// GetPageURL returns the current page URL
func (s *ITANDIScraper) GetPageURL() (string, error) {
	var url string
	err := chromedp.Run(s.ctx,
		chromedp.Location(&url),
	)
	return url, err
}

// WaitForNavigation waits for navigation to complete
func (s *ITANDIScraper) WaitForNavigation() error {
	return chromedp.Run(s.ctx,
		chromedp.WaitReady("body"),
	)
}

// closeModalAdsQuick quickly closes modal advertisements with timeout
func (s *ITANDIScraper) closeModalAdsQuick() error {
	log.Println("Quick modal check...")

	// Very short wait
	time.Sleep(500 * time.Millisecond)

	// Try direct JavaScript approach with short timeout
	jsCtx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()

	var result string
	err := chromedp.Run(jsCtx,
		chromedp.Evaluate(`
			(() => {
				try {
					// Ultra-aggressive modal closing approach
					let closed = 0;
					let actions = [];
					
					// Strategy 1: Look for any visible modal-like elements with high z-index
					const allElements = Array.from(document.querySelectorAll('*'));
					const potentialModals = allElements.filter(el => {
						const style = window.getComputedStyle(el);
						const zIndex = parseInt(style.zIndex) || 0;
						return (style.position === 'fixed' || style.position === 'absolute') &&
							   style.display !== 'none' && 
							   style.visibility !== 'hidden' &&
							   el.offsetWidth > 100 && 
							   el.offsetHeight > 100 &&
							   (zIndex > 10 || style.position === 'fixed');
					});
					
					actions.push('Found ' + potentialModals.length + ' potential modal elements');
					
					for (let modal of potentialModals) {
						// Look for ANY clickable element that might close it
						const allClickables = modal.querySelectorAll('*');
						let modalClosed = false;
						
						for (let el of allClickables) {
							const text = el.textContent ? el.textContent.trim() : '';
							const tagName = el.tagName ? el.tagName.toLowerCase() : '';
							const className = (el.className && typeof el.className === 'string') ? el.className.toLowerCase() : '';
							const title = el.title ? el.title.toLowerCase() : '';
							
							// Ultra-broad close button detection
							const isCloseButton = (text === '×' || text === '✕' || text === 'X' || text === 'x' || text === '閉じる' || text === 'close') ||
								className.includes('close') || 
								title.includes('close') || title.includes('閉じる') ||
								(tagName === 'button' && modal.contains(el)) ||
								// Look for elements positioned in top-right corner (typical close button position)
								(el.style && (el.style.right || el.style.top) && (text === '' || text.length < 3)) ||
								// SVG close icons
								(tagName === 'svg' || el.querySelector('svg')) ||
								// Elements with close-like attributes
								el.getAttribute('aria-label') === 'close' || el.getAttribute('data-close') || 
								el.getAttribute('role') === 'button';
								
							if (isCloseButton) {
								
								try {
									el.click();
									actions.push('Clicked: ' + tagName + ' with text "' + text + '"');
									modalClosed = true;
									closed++;
									break;
								} catch (e) {
									actions.push('Failed to click: ' + e.message);
								}
							}
						}
						
						// If no close button found, force hide
						if (!modalClosed) {
							modal.style.display = 'none';
							modal.style.visibility = 'hidden';
							modal.style.opacity = '0';
							modal.style.zIndex = '-9999';
							actions.push('Force hidden modal: ' + modal.tagName + '.' + modal.className);
							closed++;
						}
					}
					
					// Strategy 2: Specifically target the visible ITANDI modal close button
					const specificCloseButtons = document.querySelectorAll('div, span, button, a, svg');
					for (let btn of specificCloseButtons) {
						const style = window.getComputedStyle(btn);
						const rect = btn.getBoundingClientRect();
						
						// Look for elements in the top-right area that might be close buttons
						if (rect.right > window.innerWidth - 100 && rect.top < 100 &&
							btn.offsetWidth > 0 && btn.offsetHeight > 0 &&
							style.cursor === 'pointer') {
							try {
								btn.click();
								actions.push('Clicked potential close button at top-right: ' + btn.tagName);
								closed++;
							} catch (e) {
								actions.push('Failed to click top-right element: ' + e.message);
							}
						}
					}
					
					// Strategy 3: Remove specific ITANDI modal classes
					const itandiModals = document.querySelectorAll('[class*="go"], [class*="modal"], [id*="modal"]');
					for (let modal of itandiModals) {
						const style = window.getComputedStyle(modal);
						if (style.position === 'fixed' || style.position === 'absolute') {
							modal.remove();
							actions.push('Removed ITANDI modal element');
							closed++;
						}
					}
					
					// Also press Escape
					document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
					
					return 'Closed ' + closed + ' modals. Actions: ' + actions.join('; ');
				} catch (e) {
					return 'Error: ' + e.message;
				}
			})()
		`, &result),
	)

	if err == nil {
		log.Printf("Quick modal result: %s\n", result)
	} else {
		log.Printf("Quick modal failed: %v\n", err)
	}

	return nil
}

// closeModalAds closes any modal advertisements that might be blocking the interface
func (s *ITANDIScraper) closeModalAds() error {
	log.Println("Attempting to close modal advertisements...")

	// Give modals time to appear (reduced from 2 seconds)
	time.Sleep(1 * time.Second)

	// Try a simple, robust approach
	var modalsClosed int

	// Method 1: Try common close button selectors
	closeSelectors := []string{
		`button[aria-label="close"]`,
		`button[title="close"]`,
		`button[class*="close"]`,
		`.close-button`,
		`.modal-close`,
		`[role="dialog"] button`,
		`button:contains("×")`,
		`button:contains("✕")`,
		`span:contains("×")`,
		`a:contains("×")`,
		// ITANDI BB specific modal close patterns
		`.modal .close`,
		`[class*="modal"] [class*="close"]`,
		`div[style*="position: fixed"] button`,
		`div[style*="position: absolute"] button`,
	}

	for _, selector := range closeSelectors {
		err := chromedp.Run(s.ctx,
			chromedp.Click(selector, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Closed modal using selector: %s\n", selector)
			modalsClosed++
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Method 2: Use JavaScript to quickly find and close modals (with timeout)
	var jsResult string

	// Create a context with timeout for the JavaScript execution
	jsCtx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	err := chromedp.Run(jsCtx,
		chromedp.Evaluate(`
			(() => {
				let actions = [];
				
				try {
					// Quick scan for visible modal-like elements
					const modalSelectors = [
						'[role="dialog"]',
						'.modal',
						'[class*="modal"]', 
						'[class*="popup"]',
						'[class*="overlay"]',
						'div[style*="position: fixed"]',
						'div[style*="z-index"]'
					];
					
					let found = 0;
					let closed = 0;
					
					for (let selector of modalSelectors) {
						const elements = document.querySelectorAll(selector);
						for (let el of elements) {
							const style = window.getComputedStyle(el);
							if (style.display !== 'none' && style.visibility !== 'hidden' &&
								(style.position === 'fixed' || style.position === 'absolute') &&
								el.offsetWidth > 50 && el.offsetHeight > 50) {
								
								found++;
								
								// Try to find and click close button
								const closeBtn = el.querySelector('button, [class*="close"], [title*="close"], [aria-label*="close"]');
								if (closeBtn) {
									closeBtn.click();
									closed++;
									actions.push('Clicked close in: ' + selector);
								} else {
									// Force hide
									el.style.display = 'none';
									closed++;
									actions.push('Hid: ' + selector);
								}
							}
						}
					}
					
					actions.push('Found: ' + found + ', Closed: ' + closed);
					
				} catch (e) {
					actions.push('Error: ' + e.message);
				}
				
				return actions.join('; ');
			})()
		`, &jsResult),
	)

	if err == nil {
		log.Printf("JavaScript modal handling: %s\n", jsResult)
	} else {
		log.Printf("JavaScript modal handling failed: %v\n", err)
	}

	// Method 3: Try keyboard shortcuts
	chromedp.Run(s.ctx, chromedp.KeyEvent("\x1b")) // Escape

	// Give time for everything to settle (reduced)
	time.Sleep(500 * time.Millisecond)

	log.Printf("Modal closing attempt completed (closed %d via selectors)\n", modalsClosed)
	return nil
}

// Helper function to write file
func writeFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}
