package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/tebeka/selenium"
)

const (
	chromeDriverPath = "driver/chromedriver"
	port             = 8080
	maxPagingIndex   = 30
)

var (
	EMAIL_REGEX  = regexp.MustCompile(`(?i)[A-Z0-9._%+-]+@(?:[A-Z0-9-]+\.)+[A-Z]{2,}`)
	PHONE_NUMBER = regexp.MustCompile("[0-9]{2,3}?-?[0-9]{3,4}-?[0-9]{4}")
)

func main() {
	searchKeyword := flag.String("keyword", "", "")
	maxPagingIndex := flag.Int("max", 30, "")
	flag.Parse()

	service, err := selenium.NewChromeDriverService(chromeDriverPath, port)
	if err != nil {
		panic(err)
	}
	defer service.Stop()

	caps := selenium.Capabilities{
		"browserName": "chrome",
	}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		panic(err)
	}
	defer wd.Quit()
	wd.SetImplicitWaitTimeout(3 * time.Second)

	output, err := os.Create("./output.csv")
	if err != nil {
		panic(err)
	}
	defer output.Close()
	writer := csv.NewWriter(bufio.NewWriter(output))

	urlFormat := "https://search.shopping.naver.com/search/all?query=%v&productSet=checkout&pagingSize=80&pagingIndex=%v"
	for pagingIndex := 1; pagingIndex <= *maxPagingIndex; pagingIndex++ {
		if err := wd.Get(fmt.Sprintf(urlFormat, *searchKeyword, pagingIndex)); err != nil {
			fmt.Println(err)
			continue
		}
		scrollToBottom(wd)

		rootWindowId, _ := wd.CurrentWindowHandle()
		items, err := wd.FindElements(selenium.ByXPATH, "//div[contains(@class, 'basicList_title')]/a")
		if err != nil {
			fmt.Printf("[ERROR, find-items]: %v", err)
			continue
		}

		for _, itemElem := range items {
			itemElem.Click()
			handles, _ := wd.WindowHandles()
			for _, h := range handles {
				if h != rootWindowId {
					wd.SwitchWindow(h)
					break
				}
			}

			row := []string{}
			if companyNameElem, err := wd.FindElement(selenium.ByXPATH, "//th[contains(text(), '상호명')]/..//td"); err == nil {
				companyName, _ := companyNameElem.Text()
				row = append(row, companyName)
			}
			if locationElem, err := wd.FindElement(selenium.ByXPATH, "//th[contains(text(), '사업장소재지')]/..//td"); err == nil {
				locationInfo, _ := locationElem.Text()
				row = append(row, PHONE_NUMBER.FindString(locationInfo), EMAIL_REGEX.FindString(locationInfo))
			}
			writer.Write(row)
			writer.Flush()
			wd.Close()
			wd.SwitchWindow(rootWindowId)
			time.Sleep(time.Second)
		}
	}
}

func scrollToBottom(wd selenium.WebDriver) {
	screenHeightResult, _ := wd.ExecuteScript("return window.screen.height;", []any{})
	screenHeight := int(screenHeightResult.(float64))
	scrollHeight := 0
	i := 0
	for {
		wd.ExecuteScript(fmt.Sprintf("window.scrollTo(0, %v*%v);", scrollHeight, i), []any{})
		result, _ := wd.ExecuteScript("return document.body.scrollHeight;", []any{})
		scrollHeight = int(result.(float64))
		if screenHeight*i > scrollHeight {
			break
		}
		i += 1
		time.Sleep(1 * time.Second)
	}
}
