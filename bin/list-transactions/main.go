package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/brave-intl/bat-go/utils/altcurrency"
	"github.com/brave-intl/bat-go/utils/formatters"
	"github.com/brave-intl/bat-go/wallet"
	"github.com/brave-intl/bat-go/wallet/provider"
	log "github.com/sirupsen/logrus"
)

const (
	dateFormat = "2006-01-02T15:04:05-0700"
)

var verbose = flag.Bool("v", false, "verbose output")
var csvOut = flag.Bool("csv", false, "csv output")
var limit = flag.Int("limit", 50, "limit number of transactions returned")
var startDateStr = flag.String("start-date", "none", "only include transactions after this datetime  [ISO 8601]")
var walletProvider = flag.String("provider", "uphold", "provider for the source wallet")
var signed = flag.Bool("signed", false, "signed value depending on transaction direction")

func main() {
	log.SetFormatter(&formatters.CliFormatter{})

	flag.Usage = func() {
		log.Printf("A helper for fetching transaction history.\n\n")
		log.Printf("Usage:\n\n")
		log.Printf("        %s PROVIDER_ID\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	startDate := time.Unix(0, 0)
	if *startDateStr != "none" {
		startDate, err = time.Parse(dateFormat, *startDateStr)
		if err != nil {
			log.Fatalf("%s is not a valid ISO 8601 datetime\n", *startDateStr)
		}
	}

	walletc := altcurrency.BAT
	info := wallet.Info{
		Provider:    *walletProvider,
		ProviderID:  flag.Args()[0],
		AltCurrency: &walletc,
	}
	w, err := provider.GetWallet(context.Background(), info)
	if err != nil {
		log.Fatalln(err)
	}

	txns, err := w.ListTransactions(*limit, startDate)
	if err != nil {
		log.Fatalln(err)
	}

	sort.Sort(wallet.ByTime(txns))

	if *csvOut {
		w := csv.NewWriter(os.Stdout)
		err = w.Write([]string{"id", "date", "description", "probi", "altcurrency", "source", "destination", "transferFee", "exchangeFee", "destAmount", "destCurrency"})
		if err != nil {
			log.Fatal(err)
		}

		for i := 0; i < len(txns); i++ {
			t := txns[i]

			value := t.AltCurrency.FromProbi(t.Probi).String()
			if *signed {
				if t.Source == info.ProviderID {
					value = "-" + value
				} else if t.Destination == info.ProviderID {
					value = "+" + value
				} else {
					panic("Could not determine direction of transaction")
				}
			}

			record := []string{
				t.ID,
				t.Time.String(),
				t.Note,
				value,
				t.AltCurrency.String(),
				t.Source,
				t.Destination,
				t.TransferFee.String(),
				t.ExchangeFee.String(),
				t.DestAmount.String(),
				t.DestCurrency,
			}
			if err := w.Write(record); err != nil {

				log.Fatalln("error writing record to csv:", err)
			}
		}

		w.Flush()

		if err := w.Error(); err != nil {
			log.Fatal(err)
		}
	} else {
		for i := 0; i < len(txns); i++ {
			fmt.Printf("%s\n", txns[i])
		}
	}

}
