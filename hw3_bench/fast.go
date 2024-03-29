package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

const filePathFast string = "./data/users.txt"

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePathFast)
	if err != nil {
		panic(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	seenBrowsers := []string{}
	uniqueBrowsers := 0
	foundUsers := ""

	lines := strings.Split(string(fileContents), "\n")

	users := make([]map[string]interface{}, 0)
	for _, line := range lines {
		user := make(map[string]interface{})
		// fmt.Printf("%v %v\n", err, line)
		err := json.Unmarshal([]byte(line), &user)
		if err != nil {
			panic(err)
		}
		// fmt.Printf("%v\n", user)
		users = append(users, user)
	}

	for i, user := range users {

		isAndroid := false
		isMSIE := false

		browsers, ok := user["browsers"].([]interface{})
		if !ok {
			// log.Println("cant cast browsers")
			continue
		}

		for _, browserRaw := range browsers {
			browser, ok := browserRaw.(string)
			if !ok {
				// log.Println("cant cast browser to string")
				continue
			}

			browserFound := false

			if ok := strings.Contains(browser, "Android"); ok {
				isAndroid = true
				browserFound = true
			}

			if ok := strings.Contains(browser, "MSIE"); ok {
				isMSIE = true
				browserFound = true
			}

			if browserFound {
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		// если у пользователя оба браузера то запоминаем его
		if isAndroid && isMSIE {
			// log.Println("Android and MSIE user:", user["name"], user["email"])
			r := regexp.MustCompile("@")
			email := r.ReplaceAllString(user["email"].(string), " [at] ")
			foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user["name"], email)
		}
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}

func main() {
	FastSearch(os.Stdout)
}
