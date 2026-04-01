package main

import (
	"fmt"
	"io"
	"net/http"
)

const (
	PREVCLOSE = "/v2/aggs/ticker/"
	APIKEY    = "apiKey"
)

var (
	apikey string = "uC7pV1TTGTicMA2G5UPOMc3nk9Wqcayp"
	apiurl string = "https://api.polygon.io"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)

}

func handler(w http.ResponseWriter, r *http.Request) {
	prevClose, err := PreviousClose("AAPL")
	if err != nil {
		fmt.Println(err)
	}
	w.Write([]byte(prevClose))
}

func PreviousClose(ticker string) (result string, err error) {
	url := fmt.Sprintf("%s%s%s/prev?adjusted=true&%s=%s", apiurl, PREVCLOSE, ticker, APIKEY, apikey)

	client := &http.Client{}

	httpRequest, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return result, err
	}

	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		return result, err
	}

	resp, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return result, err
	}
	return fmt.Sprintf("%s", resp), nil
}
