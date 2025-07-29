package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// Parse command line arguments
	propertyName := flag.String("property", "", "Property name to search for")
	headless := flag.Bool("headless", false, "Run in headless mode")
	analyze := flag.Bool("analyze", false, "Run in analysis mode to inspect HTML structure")
	updated := flag.Bool("updated", false, "Use updated scraper that works with actual ITANDI BB structure")
	findLogin := flag.Bool("find-login", false, "Search for email/password login pages")
	emailLogin := flag.Bool("email-login", false, "Use email/password login instead of phone verification")
	analyzeSearch := flag.Bool("analyze-search", false, "Analyze the search flow")
	detailedAnalysis := flag.Bool("detailed-analysis", false, "Detailed analysis of search results")
	testModal := flag.Bool("test-modal", false, "Test modal advertisement handling")
	flag.Parse()

	// Run analysis mode
	if *analyze {
		runAnalysis()
		return
	}

	// Run updated scraper
	if *updated {
		runUpdatedScraper()
		return
	}

	// Find login pages
	if *findLogin {
		findLoginPage()
		return
	}

	// Email/password login
	if *emailLogin {
		runEmailLogin()
		return
	}

	// Analyze search flow
	if *analyzeSearch {
		analyzeSearchFlow()
		return
	}

	// Detailed analysis
	if *detailedAnalysis {
		analyzeDetailedSearch()
		return
	}
	
	// Test modal handling
	if *testModal {
		testModalHandling()
		return
	}

	if *propertyName == "" {
		log.Println("No property name specified. Running in demo mode.")
		*propertyName = "サンプル物件" // Default property name for testing
	}

	// Create scraper instance
	scraper, err := NewITANDIScraper(*headless)
	if err != nil {
		log.Fatal("Failed to create scraper:", err)
	}
	defer scraper.Close()

	// Step 1: Navigate to login page
	log.Println("=== Step 1: Navigating to login page ===")
	if err := scraper.NavigateToLogin(); err != nil {
		log.Fatal("Failed to navigate:", err)
	}

	// Wait a bit for page to fully load
	time.Sleep(2 * time.Second)

	// Take screenshot for verification
	if err := scraper.TakeScreenshot("step1_login_page.png"); err != nil {
		log.Println("Warning: Failed to take screenshot:", err)
	}

	log.Println("Step 1 completed: Successfully accessed ITANDI login page")
	
	// Step 2: Perform login
	log.Println("\n=== Step 2: Logging in ===")
	if err := scraper.Login(); err != nil {
		log.Fatal("Failed to login:", err)
	}
	
	// Take screenshot after login
	if err := scraper.TakeScreenshot("step2_after_login.png"); err != nil {
		log.Println("Warning: Failed to take screenshot:", err)
	}
	
	log.Println("Step 2 completed: Successfully logged in to ITANDI BB")
	
	// Step 3: Search for property
	log.Printf("\n=== Step 3: Searching for property '%s' ===\n", *propertyName)
	if err := scraper.SearchProperty(*propertyName); err != nil {
		log.Fatal("Failed to search property:", err)
	}
	
	// Take screenshot of search results
	if err := scraper.TakeScreenshot("step3_search_results.png"); err != nil {
		log.Println("Warning: Failed to take screenshot:", err)
	}
	
	log.Println("Step 3 completed: Property search executed")
	
	// Step 4: Get property details
	log.Println("\n=== Step 4: Extracting property details ===")
	details, err := scraper.GetPropertyDetails()
	if err != nil {
		log.Printf("Warning: Failed to get property details: %v\n", err)
	} else {
		// Print details in JSON format for easy parsing
		jsonData, _ := json.MarshalIndent(details, "", "  ")
		fmt.Printf("\nProperty Details (JSON):\n%s\n", jsonData)
		
		// Save to JSON file
		jsonFileName := fmt.Sprintf("property_details_%s.json", time.Now().Format("20060102_150405"))
		if err := os.WriteFile(jsonFileName, jsonData, 0644); err != nil {
			log.Printf("Error saving JSON file: %v\n", err)
		} else {
			fmt.Printf("\nJSON saved to: %s\n", jsonFileName)
		}
		
		// Also print in readable format
		fmt.Println("\nProperty Details:")
		for key, value := range details {
			fmt.Printf("- %s: %s\n", key, value)
		}
	}
	
	// Take final screenshot
	if err := scraper.TakeScreenshot("step4_property_details.png"); err != nil {
		log.Println("Warning: Failed to take screenshot:", err)
	}
	
	log.Println("\n=== All steps completed ===")
	
	// Keep browser open for a few seconds for visual confirmation if not headless
	if !*headless {
		log.Println("Keeping browser open for 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}