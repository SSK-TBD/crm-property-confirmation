package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

// EmailLoginScraper handles email/password based login
type EmailLoginScraper struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEmailLoginScraper creates a new email login scraper
func NewEmailLoginScraper(headless bool) (*EmailLoginScraper, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/Applications/Chromium.app/Contents/MacOS/Chromium"),
		chromedp.Flag("headless", headless),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel2 := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))

	combinedCancel := func() {
		cancel2()
		cancel()
	}

	return &EmailLoginScraper{
		ctx:    ctx,
		cancel: combinedCancel,
	}, nil
}

// Close cleans up resources
func (s *EmailLoginScraper) Close() {
	s.cancel()
}

// FindEmailLoginForm searches for email/password login forms across multiple strategies
func (s *EmailLoginScraper) FindEmailLoginForm() error {
	log.Println("Searching for email/password login form...")

	// Strategy 1: Check if there are hidden login forms that appear on click
	log.Println("Strategy 1: Looking for login buttons/links...")
	
	err := chromedp.Run(s.ctx,
		chromedp.Navigate("https://itandi-accounts.com/"),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		return fmt.Errorf("failed to navigate: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Look for login-related buttons or links
	var loginElements []string
	err = chromedp.Run(s.ctx,
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a, button')).filter(el => {
				const text = el.textContent.toLowerCase();
				return text.includes('ログイン') || text.includes('login') || 
					   text.includes('sign in') || text.includes('サインイン');
			}).map(el => ({
				text: el.textContent,
				href: el.href,
				className: el.className,
				id: el.id
			}))
		`, &loginElements),
	)
	
	if err == nil {
		log.Printf("Found %d login-related elements\n", len(loginElements))
	}

	// Strategy 2: Try clicking on potential login elements
	loginSelectors := []string{
		`a[href*="login"]`,
		`a[href*="sign_in"]`,
		`button:contains("ログイン")`,
		`button:contains("Login")`,
		`.login-btn`,
		`.signin-btn`,
		`#login-button`,
		`#signin-button`,
	}

	for _, selector := range loginSelectors {
		log.Printf("Trying selector: %s\n", selector)
		err = chromedp.Run(s.ctx,
			chromedp.Click(selector, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Clicked on: %s\n", selector)
			time.Sleep(3 * time.Second)
			
			// Check if email/password inputs appeared
			if s.hasEmailPasswordInputs() {
				log.Printf("✅ Found email/password form after clicking: %s\n", selector)
				return nil
			}
		}
	}

	// Strategy 3: Check for modal dialogs or overlays
	log.Println("Strategy 3: Looking for modal dialogs...")
	
	modalSelectors := []string{
		`.modal input[type="email"]`,
		`.overlay input[type="email"]`,
		`.popup input[type="email"]`,
		`.dialog input[type="email"]`,
	}

	for _, selector := range modalSelectors {
		var found bool
		chromedp.Run(s.ctx,
			chromedp.EvaluateAsDevTools(fmt.Sprintf(`document.querySelector('%s') !== null`, selector), &found),
		)
		if found {
			log.Printf("✅ Found email input in modal: %s\n", selector)
			return nil
		}
	}

	// Strategy 4: Try common login URLs with different approaches
	alternativeURLs := []string{
		"https://accounts.itandi.com/",
		"https://login.itandi.com/",
		"https://auth.itandi.com/",
		"https://itandi.com/login",
		"https://app.itandi.com/login",
		"https://bukkakun.com/auth/login",
	}

	for _, url := range alternativeURLs {
		log.Printf("Trying alternative URL: %s\n", url)
		err = chromedp.Run(s.ctx,
			chromedp.Navigate(url),
			chromedp.WaitReady("body"),
		)
		
		if err == nil {
			time.Sleep(2 * time.Second)
			if s.hasEmailPasswordInputs() {
				log.Printf("✅ Found email/password form at: %s\n", url)
				return nil
			}
		}
	}

	return fmt.Errorf("no email/password login form found")
}

// hasEmailPasswordInputs checks if the current page has email and password inputs
func (s *EmailLoginScraper) hasEmailPasswordInputs() bool {
	var hasEmail, hasPassword bool
	
	chromedp.Run(s.ctx,
		chromedp.EvaluateAsDevTools(`
			document.querySelector('input[type="email"], input[name="email"], input[id*="email"], input[placeholder*="email"], input[placeholder*="メール"]') !== null
		`, &hasEmail),
	)
	
	chromedp.Run(s.ctx,
		chromedp.EvaluateAsDevTools(`
			document.querySelector('input[type="password"], input[name="password"], input[id*="password"], input[placeholder*="password"], input[placeholder*="パスワード"]') !== null
		`, &hasPassword),
	)
	
	log.Printf("Email input found: %v, Password input found: %v\n", hasEmail, hasPassword)
	return hasEmail && hasPassword
}

// PerformEmailLogin performs login with email and password
func (s *EmailLoginScraper) PerformEmailLogin(email, password string) error {
	log.Printf("Attempting login with email: %s\n", email)

	// Find and fill email input
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
			chromedp.SendKeys(selector, email, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Email entered using selector: %s\n", selector)
			emailFilled = true
			break
		}
	}

	if !emailFilled {
		return fmt.Errorf("could not find email input field")
	}

	// Find and fill password input
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
			chromedp.SendKeys(selector, password, chromedp.ByQuery, chromedp.AtLeast(0)),
		)
		if err == nil {
			log.Printf("Password entered using selector: %s\n", selector)
			passwordFilled = true
			break
		}
	}

	if !passwordFilled {
		return fmt.Errorf("could not find password input field")
	}

	// Submit the form
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
			log.Printf("Submit button clicked: %s\n", selector)
			submitted = true
			break
		}
	}

	if !submitted {
		// Try pressing Enter as fallback
		err := chromedp.Run(s.ctx,
			chromedp.KeyEvent("\r"),
		)
		if err == nil {
			log.Println("Form submitted using Enter key")
			submitted = true
		}
	}

	if !submitted {
		return fmt.Errorf("could not submit login form")
	}

	// Wait for navigation
	time.Sleep(5 * time.Second)
	
	log.Println("Login form submitted successfully")
	return nil
}

// TakeScreenshot takes a screenshot
func (s *EmailLoginScraper) TakeScreenshot(filename string) error {
	var buf []byte
	
	err := chromedp.Run(s.ctx,
		chromedp.CaptureScreenshot(&buf),
	)
	
	if err != nil {
		return fmt.Errorf("failed to take screenshot: %w", err)
	}
	
	if err := os.WriteFile(filename, buf, 0644); err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}
	
	log.Printf("Screenshot saved to %s\n", filename)
	return nil
}

// GetCurrentURL returns current URL
func (s *EmailLoginScraper) GetCurrentURL() (string, error) {
	var url string
	err := chromedp.Run(s.ctx,
		chromedp.Location(&url),
	)
	return url, err
}