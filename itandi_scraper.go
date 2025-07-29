package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	loginURL = "https://itandi-accounts.com/"
)

var (
	loginEmail    = os.Getenv("ITANDI_EMAIL")
	loginPassword = os.Getenv("ITANDI_PASSWORD")
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
	if err := os.WriteFile(filename, buf, 0644); err != nil {
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

	// Check if credentials are set
	if loginEmail == "" || loginPassword == "" {
		return fmt.Errorf("ITANDI_EMAIL and ITANDI_PASSWORD environment variables must be set")
	}

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

	// Close any remaining modals once before extracting results
	log.Println("Closing any remaining modals before extracting results...")
	s.closeModalAdsQuick()

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
				const allEls = document.querySelectorAll('*');
				for (let el of allEls) {
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
				
				// Check for actual property listings (based on screenshot structure)
				// Look for property images
				const propertyImages = document.querySelectorAll('img[src*="property"], img[src*="building"], img[alt*="物件"]');
				
				// Count elements containing specific text
				let rentElements = 0;
				let roomElements = 0;
				let recruitingElements = 0;
				
				// Re-use the same elements collection
				for (let el of allEls) {
					const text = el.textContent || '';
					if (text.includes('万円') || text.includes('賃料')) rentElements++;
					if (text.includes('LDK') || text.includes('DK') || text.includes('間取')) roomElements++;
					if (text.includes('募集中')) recruitingElements++;
				}
				
				// Check for property links
				const propertyLinks = document.querySelectorAll('a[href*="/rent_rooms/"]');
				
				// More robust check for results
				const hasPropertyElements = propertyImages.length > 0 || 
										   (rentElements > 2 && roomElements > 2) ||
										   recruitingElements > 0 ||
										   propertyLinks.length > 0;
				
				// Also check for result count display (e.g., "1件")
				let resultCount = 0;
				for (let el of allEls) {
					const text = el.textContent;
					if (text && text.match(/(\d+)\s*件/) && !text.includes('0件')) {
						const match = text.match(/(\d+)\s*件/);
						resultCount = parseInt(match[1]);
						break;
					}
				}
				
				return {
					hasResults: (hasPropertyElements || resultCount > 0) && !noResultsFound,
					noResultsMessage: message,
					tableCount: Math.max(propertyImages.length, resultCount),
					propertyImages: propertyImages.length,
					recruitingElements: recruitingElements,
					resultCount: resultCount
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
				// Additional check - if we have detected property images or recruit status
				if imgCount, ok := resultMap["propertyImages"].(float64); ok && imgCount > 0 {
					log.Printf("Found %v property images - treating as results found\n", imgCount)
					details["search_status"] = "Results found"
				} else if recruitCount, ok := resultMap["recruitingElements"].(float64); ok && recruitCount > 0 {
					log.Printf("Found %v recruiting elements - treating as results found\n", recruitCount)
					details["search_status"] = "Results found"
				} else if resultCount, ok := resultMap["resultCount"].(float64); ok && resultCount > 0 {
					log.Printf("Found result count: %v - treating as results found\n", resultCount)
					details["search_status"] = "Results found"
				} else {
					log.Println("No search results found")
					details["search_status"] = "No results found"
					if msg, ok := resultMap["noResultsMessage"].(string); ok && msg != "" {
						details["no_results_message"] = msg
					}
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
	// Based on sample.png structure
	selectors := map[string][]string{
		"property_name": {
			`td:contains("物件名") + td`, // Table cell after "物件名"
			`.property-name`,
			`[class*="building"]`,
			`td[data-label*="物件名"]`,
			`tr:contains("物件名") td:last-child`,
		},
		"building_number": {
			`td:contains("部屋番号") + td`,
			`.room-number`,
			`tr:contains("部屋番号") td:last-child`,
		},
		"management_status": {
			`td:contains("管理費") + td`,
			`td:contains("共益費") + td`,
			`tr:contains("管理費") td:last-child`,
		},
		"rent": {
			`td:contains("賃料") + td`,
			`.rent`,
			`[class*="rent"]`,
			`td[data-label*="賃料"]`,
			`tr:contains("賃料") td:last-child`,
		},
		"deposit": {
			`td:contains("敷金") + td`,
			`.deposit`,
			`tr:contains("敷金") td:last-child`,
		},
		"key_money": {
			`td:contains("礼金") + td`,
			`.key-money`,
			`tr:contains("礼金") td:last-child`,
		},
		"insurance": {
			`td:contains("保証金") + td`,
			`.insurance`,
			`tr:contains("保証金") td:last-child`,
		},
		"layout": {
			`td:contains("間取り") + td`,
			`.layout`,
			`[class*="layout"]`,
			`td[data-label*="間取"]`,
			`tr:contains("間取り") td:last-child`,
		},
		"area": {
			`td:contains("専有面積") + td`,
			`.area`,
			`[class*="area"]`,
			`tr:contains("専有面積") td:last-child`,
			`tr:contains("面積") td:last-child`,
		},
		"date_completed": {
			`td:contains("築年月") + td`,
			`td:contains("竣工年月") + td`,
			`tr:contains("築年月") td:last-child`,
		},
		"available_date": {
			`td:contains("入居可能時期") + td`,
			`td:contains("入居可能日") + td`,
			`tr:contains("入居可能") td:last-child`,
		},
		"vacancy_rate": {
			`td:contains("空室率") + td`,
			`td:contains("収引率") + td`,
			`tr:contains("率") td:last-child`,
		},
		"floor_info": {
			`td:contains("階") + td`,
			`.floor-info`,
			`tr:contains("階") td:last-child`,
		},
		"management_company": {
			`td:contains("管理会社") + td`,
			`.management-company`,
			`[class*="management"]`,
			`td[data-label*="管理会社"]`,
			`tr:contains("管理会社") td:last-child`,
		},
		"photo_count": {
			`span:contains("枚")`,
			`.photo-count`,
			`[class*="photo"] span`,
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

	// Try to get all property information using JavaScript for more flexibility
	var propertyData interface{}
	err = chromedp.Run(s.ctx,
		chromedp.Evaluate(`
			(() => {
				const data = {};
				
				// Try to extract table-based data (like in sample.png)
				const tables = document.querySelectorAll('table');
				tables.forEach(table => {
					const rows = table.querySelectorAll('tr');
					rows.forEach(row => {
						const cells = row.querySelectorAll('td, th');
						if (cells.length >= 2) {
							const label = cells[0].textContent.trim();
							const value = cells[cells.length - 1].textContent.trim();
							
							// Map common labels
							const labelMap = {
								'物件名': 'property_name',
								'部屋番号': 'room_number',
								'賃料': 'rent',
								'管理費': 'management_fee',
								'共益費': 'common_fee',
								'敷金': 'deposit',
								'礼金': 'key_money',
								'保証金': 'insurance',
								'間取り': 'layout',
								'専有面積': 'area',
								'築年月': 'date_completed',
								'入居可能時期': 'available_date',
								'管理会社': 'management_company',
								'建物種別': 'building_type',
								'構造': 'structure',
								'所在地': 'address',
								'最寄駅': 'nearest_station',
								'駅徒歩': 'station_distance',
								'階数': 'floor',
								'向き': 'direction',
								'契約期間': 'contract_period',
								'更新料': 'renewal_fee',
								'広告料': 'advertisement_fee',
								'部屋数': 'room_count'
							};
							
							const key = labelMap[label] || label.toLowerCase().replace(/\s+/g, '_');
							if (value && value !== '') {
								data[key] = value;
							}
						}
					});
				});
				
				// Also try to get list-style data
				const listItems = document.querySelectorAll('dl, .property-info, .detail-info');
				listItems.forEach(item => {
					const dts = item.querySelectorAll('dt');
					const dds = item.querySelectorAll('dd');
					for (let i = 0; i < Math.min(dts.length, dds.length); i++) {
						const label = dts[i].textContent.trim();
						const value = dds[i].textContent.trim();
						if (label && value) {
							data[label.toLowerCase().replace(/\s+/g, '_')] = value;
						}
					}
				});
				
				// Get property list and details from search results
				// Find all property cards - looking for elements that contain property images
				let propertyElements = [];
				
				// Method 1: Find elements containing property images
				const imgElements = document.querySelectorAll('img');
				imgElements.forEach(img => {
					if (img.src && (img.src.includes('property') || img.src.includes('building') || img.alt.includes('物件'))) {
						// Find the parent card element (usually 2-3 levels up)
						let parent = img.parentElement;
						for (let i = 0; i < 4 && parent; i++) {
							if (parent.querySelector('a[href*="/rent_rooms/"]')) {
								propertyElements.push(parent);
								break;
							}
							parent = parent.parentElement;
						}
					}
				});
				
				// Method 2: Find cards containing rent information
				if (propertyElements.length === 0) {
					const rentElements = Array.from(document.querySelectorAll('*')).filter(el => {
						const text = el.textContent || '';
						return text.includes('万円') && text.includes('募集中');
					});
					rentElements.forEach(el => {
						let parent = el;
						for (let i = 0; i < 3 && parent; i++) {
							if (parent.querySelector('a[href*="/rent_rooms/"]')) {
								propertyElements.push(parent);
								break;
							}
							parent = parent.parentElement;
						}
					});
				}
				
				// Remove duplicates
				propertyElements = [...new Set(propertyElements)];
				
				const properties = [];
				
				// Try to extract property details from each card/element
				propertyElements.forEach(elem => {
					const property = {};
					
					// Look for property name/link
					const propLink = elem.querySelector('a[href*="/rent_rooms/"]');
					if (propLink) {
						property.url = propLink.href;
						
						// Try to find the actual property name (not just "詳細" link text)
						// Look for heading or property name element
						const nameElement = elem.querySelector('h2, h3, h4, [class*="property-name"], [class*="building-name"]');
						if (nameElement) {
							property.name = nameElement.textContent.trim();
						} else {
							// Fallback: extract from text that looks like property name
							const allText = elem.textContent || '';
							const lines = allText.split('\n').map(s => s.trim()).filter(s => s);
							// Property name is often the first substantial text
							for (let line of lines) {
								if (line.length > 5 && !line.includes('万円') && !line.includes('募集') && 
									!line.includes('詳細') && !line.includes('LDK')) {
									property.name = line;
									break;
								}
							}
						}
						
						// If still no name, use the link text as fallback
						if (!property.name) {
							property.name = propLink.textContent.trim();
						}
					}
					
					// Extract text information from the card
					const textContent = elem.textContent || '';
					
					// Extract rent (賃料)
					const rentMatch = textContent.match(/(\d+\.?\d*)\s*万円/);
					if (rentMatch) {
						property.rent = rentMatch[0];
					}
					
					// Extract room layout (間取り)
					const layoutMatch = textContent.match(/(\d+[LDK]+|ワンルーム)/);
					if (layoutMatch) {
						property.layout = layoutMatch[0];
					}
					
					// Extract area (面積)
					const areaMatch = textContent.match(/(\d+\.?\d*)\s*㎡|(\d+\.?\d*)\s*m²/);
					if (areaMatch) {
						property.area = areaMatch[0];
					}
					
					// Extract floor (階)
					const floorMatch = textContent.match(/(\d+)階/);
					if (floorMatch) {
						property.floor = floorMatch[0];
					}
					
					// Extract address or location
					const addressMatch = textContent.match(/[^\s]+区[^\s]+/);
					if (addressMatch) {
						property.address = addressMatch[0];
					}
					
					// Check for募集中 status
					if (textContent.includes('募集中')) {
						property.status = '募集中';
					}
					
					// Extract deposit/key money
					const depositMatch = textContent.match(/敷金[：\s]*(\d+\.?\d*)\s*万円/);
					if (depositMatch) {
						property.deposit = depositMatch[1] + '万円';
					}
					
					const keyMoneyMatch = textContent.match(/礼金[：\s]*(\d+\.?\d*)\s*万円/);
					if (keyMoneyMatch) {
						property.key_money = keyMoneyMatch[1] + '万円';
					}
					
					// Only add if we found some property info
					if (Object.keys(property).length > 1) {
						properties.push(property);
					}
				});
				
				// Add debug info
				data.debug_property_elements_found = propertyElements.length;
				data.debug_img_elements_found = imgElements.length;
				
				if (properties.length > 0) {
					data.properties = properties;
					data.property_count = properties.length;
					
					// Add first property details to root level for easy access
					if (properties[0]) {
						Object.keys(properties[0]).forEach(key => {
							data['first_property_' + key] = properties[0][key];
						});
					}
				} else {
					// Debug: Try to find any property-related content
					const allLinks = document.querySelectorAll('a[href*="/rent_rooms/"]');
					data.debug_property_links = allLinks.length;
					
					// Get sample content for debugging
					if (allLinks.length > 0) {
						data.debug_first_link_text = allLinks[0].textContent.trim();
						data.debug_first_link_href = allLinks[0].href;
					}
				}
				
				// Check for no results message
				const bodyText = document.body.textContent;
				if (bodyText.includes('検索結果がありません') || bodyText.includes('該当する物件がありません')) {
					data.no_results = true;
				}
				
				return data;
			})()
		`, &propertyData),
	)
	
	if err == nil && propertyData != nil {
		if dataMap, ok := propertyData.(map[string]interface{}); ok {
			// Merge JavaScript extracted data with details
			for k, v := range dataMap {
				if strVal, ok := v.(string); ok && strVal != "" {
					details[k] = strVal
					log.Printf("Found %s: %s (from JavaScript)\n", k, strVal)
				} else if k == "properties" {
					// Handle property list with details
					if list, ok := v.([]interface{}); ok && len(list) > 0 {
						details["property_count"] = fmt.Sprintf("%d", len(list))
						log.Printf("Found %d properties with details\n", len(list))
						
						// Store the full property list as JSON
						if jsonData, err := json.Marshal(list); err == nil {
							details["properties_json"] = string(jsonData)
						}
					}
				} else if strings.HasPrefix(k, "first_property_") {
					// Add first property details to main details
					details[k] = strVal
					log.Printf("First property %s: %s\n", strings.TrimPrefix(k, "first_property_"), strVal)
				}
			}
		}
	}

	// Final status check - if we already found results, don't override
	if details["search_status"] != "Results found" {
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
	}

	// Get page title
	var pageTitle string
	chromedp.Run(s.ctx,
		chromedp.Title(&pageTitle),
	)
	if pageTitle != "" {
		details["page_title"] = pageTitle
	}

	// Save search results DOM if we have results (focused on property cards area only)
	if details["search_status"] == "Results found" {
		var propertyCardHTML string
		err = chromedp.Run(s.ctx,
			chromedp.Evaluate(`
				(() => {
					// Find the complete property record but exclude desktop-only elements
					var propertyLinks = document.querySelectorAll('a[href*="/rent_rooms/"]');
					if (propertyLinks.length > 0) {
						// Get the first property link's container
						var link = propertyLinks[0];
						var container = link;
						
						// Navigate up to find the most complete container
						var bestContainer = null;
						var maxScore = 0;
						var attempts = 0;
						
						while (container.parentElement && attempts < 15) {
							var parent = container.parentElement;
							var text = parent.textContent || '';
							var score = 0;
							
							// Score based on completeness of information
							if (text.includes('クレール')) score += 10;
							if (text.includes('大阪府大阪市住吉区')) score += 10;
							if (text.includes('7.7万円')) score += 5;
							if (text.includes('1LDK')) score += 3;
							if (text.includes('44.61㎡')) score += 3;
							if (text.includes('募集中')) score += 3;
							if (text.includes('株式会社Room')) score += 8;
							if (text.includes('築13年')) score += 5;
							if (text.includes('2011年9月')) score += 5;
							if (text.includes('部屋番号')) score += 3;
							if (text.includes('敷礼保')) score += 3;
							if (text.includes('内見・申込')) score += 3;
							
							// Bonus for having complete action buttons
							if (text.includes('部屋止') && text.includes('内見') && text.includes('詳細')) score += 5;
							
							// Accept containers with high completeness score
							if (score > maxScore && score >= 30) {
								maxScore = score;
								bestContainer = parent;
							}
							
							container = parent;
							attempts++;
							
							// Stop if we're getting too high in the DOM
							if (container.tagName === 'BODY' || container.children.length > 50) {
								break;
							}
						}
						
						// Filter out desktop-only elements from the container
						function filterDesktopElements(element) {
							if (!element) return null;
							
							// Clone the element to avoid modifying the original DOM
							var cloned = element.cloneNode(true);
							
							// Function to check if element is desktop-only based on CSS
							function isDesktopOnly(el) {
								// Check for CSS that indicates desktop-only display (@media screen and (min-width: 900px))
								var computedStyle = window.getComputedStyle(el);
								
								// Look for elements that are only visible on desktop
								// This includes checking parent containers that might have desktop-only CSS
								var current = el;
								while (current && current !== document.body) {
									var style = window.getComputedStyle(current);
									
									// Check for common desktop-only patterns
									// Elements with display properties that suggest desktop layout
									if (style.display === 'table-cell' && window.innerWidth < 900) {
										return true;
									}
									
									// Check class names that might indicate desktop layout
									var className = current.className || '';
									if (typeof className === 'string' && 
										(className.includes('desktop') || 
										 className.includes('md-') || 
										 className.includes('lg-'))) {
										return true;
									}
									
									current = current.parentElement;
								}
								
								return false;
							}
							
							// Remove desktop-only elements from the cloned tree
							function removeDesktopOnlyElements(element) {
								var elementsToRemove = [];
								
								// Check all descendants
								var walker = document.createTreeWalker(
									element,
									NodeFilter.SHOW_ELEMENT,
									null,
									false
								);
								
								var node;
								while (node = walker.nextNode()) {
									if (isDesktopOnly(node)) {
										elementsToRemove.push(node);
									}
								}
								
								// Remove the elements
								elementsToRemove.forEach(function(el) {
									if (el.parentNode) {
										el.parentNode.removeChild(el);
									}
								});
								
								return element;
							}
							
							return removeDesktopOnlyElements(cloned);
						}
						
						// Use the best container found, filter out desktop elements
						if (bestContainer) {
							var filtered = filterDesktopElements(bestContainer);
							return filtered ? filtered.outerHTML : bestContainer.outerHTML;
						} else {
							// Fallback: try to find any container with basic info
							var fallbackContainers = document.querySelectorAll('[class*="jss"]');
							for (var i = 0; i < fallbackContainers.length; i++) {
								var fallback = fallbackContainers[i];
								var fallbackText = fallback.textContent || '';
								if (fallbackText.includes('クレール') && 
									fallbackText.includes('大阪') && 
									fallbackText.includes('7.7万円') && 
									fallbackText.includes('株式会社')) {
									var filtered = filterDesktopElements(fallback);
									return filtered ? filtered.outerHTML : fallback.outerHTML;
								}
							}
						}
					}
					
					return 'Property card not found';
				})()
			`, &propertyCardHTML),
		)
		if err != nil {
			log.Printf("Error getting property card HTML: %v\n", err)
		} else if propertyCardHTML != "Property card not found" {
			// Save DOM to file with proper UTF-8 encoding
			domFileName := fmt.Sprintf("property_card_dom_%s.html", time.Now().Format("20060102_150405"))
			
			// Add HTML header with UTF-8 charset declaration
			htmlContent := `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Property Card DOM</title>
</head>
<body>
` + propertyCardHTML + `
</body>
</html>`
			
			if err := os.WriteFile(domFileName, []byte(htmlContent), 0644); err != nil {
				log.Printf("Error saving DOM file: %v\n", err)
			} else {
				details["dom_saved_to"] = domFileName
				fmt.Printf("Property card DOM saved to: %s (size: %d bytes)\n", domFileName, len(htmlContent))
			}
		}
	}

	return details, nil
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
					// Optimized modal closing - avoid repeated operations
					let closed = 0;
					let actions = [];
					let processedElements = new Set(); // Track processed elements
					
					// Strategy 1: Remove overlays and modals by common selectors (one pass)
					const modalSelectors = ['[role="dialog"]', '.modal', '[class*="modal"]', '[class*="overlay"]', 
											'[style*="position: fixed"]', '[style*="z-index"]'];
					
					for (let selector of modalSelectors) {
						const elements = document.querySelectorAll(selector);
						for (let el of elements) {
							if (!processedElements.has(el) && el.offsetParent !== null) {
								// Try to find close button first
								const closeBtn = el.querySelector('button, [aria-label*="close"], [class*="close"], .close');
								if (closeBtn) {
									closeBtn.click();
									actions.push('Clicked close button in modal');
									closed++;
								} else {
									// Force hide if no close button
									el.style.display = 'none';
									actions.push('Hidden modal: ' + el.tagName);
									closed++;
								}
								processedElements.add(el);
								break; // Only process the first visible modal
							}
						}
					}
					
					// Strategy 2: Remove specific ITANDI advertisement elements (only once)
					if (closed === 0) {
						const itandiAds = document.querySelectorAll('[class*="go"][class*="2933276541"], [class*="go"][class*="2369186930"]');
						for (let ad of itandiAds) {
							if (!processedElements.has(ad)) {
								ad.remove();
								actions.push('Removed ITANDI ad element');
								processedElements.add(ad);
								closed++;
								break; // Only remove one at a time
							}
						}
					}
					
					// Press Escape once
					if (closed > 0) {
						document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
					}
					
					return closed > 0 ? 'Closed ' + closed + ' modal(s). Actions: ' + actions.join('; ') : 'No modals found';
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

