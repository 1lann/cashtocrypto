package main

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jszwec/csvutil"
)

type CashAppRow struct {
	TransactionID   string  `csv:"Transaction ID"`
	Date            string  `csv:"Date"`
	TransactionType string  `csv:"Transaction Type"`
	Currency        string  `csv:"Currency"`
	Amount          string  `csv:"Amount"`
	Fee             string  `csv:"Fee"`
	NetAmount       string  `csv:"Net Amount"`
	AssetType       string  `csv:"Asset Type"`
	AssetPrice      string  `csv:"Asset Price"`
	AssetAmount     float64 `csv:"Asset Amount"`
	Status          string  `csv:"Status"`
	Notes           string  `csv:"Notes"`
	EntityName      string  `csv:"Name of sender/receiver"`
	Account         string  `csv:"Account"`
}

func (c *CashAppRow) ParseDollar(dollar string) (float64, error) {
	dollar = strings.ReplaceAll(dollar, "$", "")
	dollar = strings.ReplaceAll(dollar, ",", "")
	return strconv.ParseFloat(dollar, 64)
}

func (c *CashAppRow) Convert() (*CryptoComTaxRow, error) {
	t, err := time.Parse("2006-01-02 15:04:05 MST", c.Date)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse date")
	}
	amount, err := c.ParseDollar(c.Amount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse amount")
	}
	fee, err := c.ParseDollar(c.Fee)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse fee")
	}
	netAmount, err := c.ParseDollar(c.NetAmount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse net amount")
	}
	assetPrice, err := c.ParseDollar(c.AssetPrice)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse asset price")
	}

	txType := ""
	switch c.TransactionType {
	case "Bitcoin Sale":
		txType = "sell"
	case "Bitcoin Boost":
		txType = "rebate"
	default:
		return nil, errors.Errorf("unknown transaction type: %s", c.TransactionType)
	}

	row := &CryptoComTaxRow{
		Date: t.Format("01/02/2006 15:04:05"),
		Type: txType,
	}

	if c.TransactionType == "Bitcoin Boost" {
		row.ReceivedCurrency = "BTC"
		row.ReceivedAmount = EncodeNumber(c.AssetAmount)
		row.ReceivedNetWorth = EncodeNumber(netAmount)
	} else {
		row.ReceivedCurrency = "USD"
		row.ReceivedAmount = EncodeNumber(netAmount)
		row.FeeCurrency = "USD"
		row.FeeAmount = EncodeNumber(fee * -1)
		row.SentCurrency = "BTC"
		row.SentAmount = EncodeNumber(c.AssetAmount)
	}

	_, _ = assetPrice, amount

	return row, nil
}

type CryptoComTaxRow struct {
	Date             string `csv:"Date"`
	Type             string `csv:"Type"`
	ReceivedCurrency string `csv:"Received Currency"`
	ReceivedAmount   string `csv:"Received Amount"`
	ReceivedNetWorth string `csv:"Received Net Worth"`
	SentCurrency     string `csv:"Sent Currency"`
	SentAmount       string `csv:"Sent Amount"`
	SentNetWorth     string `csv:"Sent Net Worth"`
	FeeCurrency      string `csv:"Fee Currency"`
	FeeAmount        string `csv:"Fee Amount"`
	FeeNetWorth      string `csv:"Fee Net Worth"`
}

func EncodeNumber(num float64) string {
	return strconv.FormatFloat(num, 'f', -1, 64)
}

func main() {
	f, err := os.Open("./cash_app_report_btc.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	dec, err := csvutil.NewDecoder(csv.NewReader(f))
	if err != nil {
		panic(err)
	}

	of, err := os.Create("./out.csv")
	if err != nil {
		panic(err)
	}
	defer of.Close()

	w := csv.NewWriter(of)
	defer w.Flush()

	enc := csvutil.NewEncoder(w)

	for {
		var cashAppRow CashAppRow
		err := dec.Decode(&cashAppRow)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			panic(err)
		}

		row, err := cashAppRow.Convert()
		if err != nil {
			panic(err)
		}

		err = enc.Encode(row)
		if err != nil {
			panic(err)
		}
	}
}
