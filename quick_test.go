package main

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func quickTestFunc() {
	log.Println("=== Quick Modal Test ===")

	// Create a simple browser context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/Applications/Chromium.app/Contents/MacOS/Chromium"),
		chromedp.Flag("headless", false),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Navigate directly to search page to see the modal
	log.Println("Navigating directly to search page...")
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://itandibb.com/rent_rooms/list"),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		log.Printf("Navigation failed (expected): %v\n", err)
		log.Println("This is expected if not logged in - just testing modal detection")
	}

	time.Sleep(3 * time.Second)

	// Take screenshot to see current state
	var buf []byte
	err = chromedp.Run(ctx,
		chromedp.CaptureScreenshot(&buf),
	)
	if err == nil {
		writeFile("quick_test_initial.png", buf)
		log.Println("Screenshot saved: quick_test_initial.png")
	}

	// Test modal detection
	var result map[string]interface{}
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				// Look for any visible elements that might be modals
				const allElements = document.querySelectorAll('*');
				const modals = [];
				
				for (let el of allElements) {
					const style = window.getComputedStyle(el);
					if ((style.position === 'fixed' || style.position === 'absolute') &&
						style.display !== 'none' && style.visibility !== 'hidden' &&
						el.offsetWidth > 100 && el.offsetHeight > 100 &&
						parseInt(style.zIndex) > 100) {
						
						modals.push({
							tagName: el.tagName,
							className: el.className,
							id: el.id,
							zIndex: style.zIndex,
							width: el.offsetWidth,
							height: el.offsetHeight,
							innerHTML: el.innerHTML.substring(0, 200)
						});
					}
				}
				
				return {
					total: allElements.length,
					modalCount: modals.length,
					modals: modals.slice(0, 5) // First 5 only
				};
			})()
		`, &result),
	)

	if err == nil {
		log.Printf("Element analysis: %+v\n", result)
	} else {
		log.Printf("Analysis failed: %v\n", err)
	}

	log.Println("Keeping browser open for manual inspection...")
	time.Sleep(30 * time.Second)
}