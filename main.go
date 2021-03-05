package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/bundle"
	"github.com/iotaledger/iota.go/consts"
)

var iotaAPI *api.API

const defaultNode = "https://nodes.iota.org:443"
const seedLen = 81
const addrsPerBatch = 10

func main() {
	fmt.Println("\nThis program will list all addresses of your seed with a positive balance and will let you move the funds of a specific address.\n")
	getAPI()
	seed := getSeed()

	for {
		addrs, balances := getAddressesWithBalance(seed)
		printAddrWithBalance(addrs, balances)
		moveBalance(seed, addrs, balances)
		fmt.Println("Do you want to move funds of another address of this seed? (y/n): ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" {
			break
		}
	}
}

func getAPI() {

	nodeURL := defaultNode
	for {
		var err error
		iotaAPI, err = api.ComposeAPI(api.HTTPClientSettings{URI: nodeURL})
		if err == nil {
			break
		}
		nodeURL = getAltNodeURL(nodeURL)
	}
}

func getAltNodeURL(oldNode string) string {
	var URL string
	fmt.Printf("Error: Could not connect to node %s\nPlease enter a new node address in this format: https://nodes.iota.org:443\n", oldNode)
	fmt.Print("Enter node address:")
	fmt.Scanln(&URL)
	return URL
}

func getSeed() string {
	var seed string
	var answer string
	for {
		fmt.Print("\nEnter seed: ")
		fmt.Scanln(&seed)
		if !hasInvalidChars(seed) {
			if !hasInvalidChars(seed) {
				if len(seed) == seedLen {
					return seed
				}
				if len(seed) < seedLen {
					fmt.Printf("The seed has less than %d characters. Do you want to continue with this seed? (y/n): ", seedLen)
					seed = seed + strings.Repeat("9", seedLen-len(seed))
				}
				if len(seed) > seedLen {
					fmt.Printf("The seed has more than %d characters. Do you want to continue with this seed? (y/n): ", seedLen)
					seed = seed[0:81]
				}
				fmt.Scanln(&answer)
				fmt.Println()
				if answer == "y" {
					return seed
				}
			}
		} else {
			fmt.Println("\nValid seeds only contain upper case letters A-Z and the number 9.")
		}
	}

	return seed
}

func hasInvalidChars(seed string) bool {
	for _, r := range seed {
		if (r < 'A' || r > 'Z') && r != '9' {
			return true
		}
	}
	return false

}

func getAddressesWithBalance(seed string) ([]string, []uint64) {
	var addrs []string
	var balances []uint64
	var total uint64
	var answer string
	index := 0

	for {
		fmt.Printf("generating addresses #%d to #%d\n", index, index+addrsPerBatch)
		addrBatch := generateAddresses(seed, index, addrsPerBatch)
		addrs = append(addrs, addrBatch...)
		balancesBatch := getBalances(addrBatch)
		balances = append(balances, balancesBatch...)
		total += getSumBalance(balancesBatch)
		if total > 0 {
			fmt.Printf("Found a total of %di on the first %d addresses.\nIs the total balance correct? (y/n):", total, index+addrsPerBatch)
			fmt.Scanln(&answer)
			if answer == "y" {
				fmt.Println()
				break
			}
		}
		index += addrsPerBatch
	}
	return addrs, balances

}

func getSumBalance(balances []uint64) uint64 {
	var sum uint64
	for _, b := range balances {
		sum += b
	}
	return sum
}

func generateAddresses(seed string, start, count int) []string {
	var addrs []string
	for i := 0; i < count; i++ {
		addr, err := address.GenerateAddress(seed, uint64(start+i), consts.SecurityLevelMedium, true)
		if err != nil {
			panic(err)
		}
		addrs = append(addrs, addr)
	}
	return addrs
}

func getBalances(addrs []string) []uint64 {
	balances, err := iotaAPI.GetBalances(addrs)
	if err != nil {
		// handle error
		panic(err)
	}
	return balances.Balances
}

func printAddrWithBalance(addrs []string, balances []uint64) {
	var total uint64
	for i, b := range balances {
		if b > 0 {
			fmt.Printf("address #%d: %di (%s)\n", i, b, addrs[i])
			total += b

		}

	}
	fmt.Printf("Total balance: %di\n", total)
}

func moveBalance(seed string, addrs []string, balances []uint64) {
	target := "BSDAIGKOGJFQPXEKPCXJYIUEB9MPLYHEJHWOSAJOYKMPTQFN9RHBQFAC9KJFS9PZVCGKZVXFJFTVFLDGZDBVYNDOKX"
	index := getIndex(addrs, balances)
	sendBalance(seed, addrs, balances, index, target)
}

func getIndex(addrs []string, balances []uint64) int {
	var index int
	for {
		var input string
		fmt.Print("You can move the funds of an address by entering its index number.\nAddress index:")
		fmt.Scanln(&input)
		var err error
		index, err = strconv.Atoi(input)
		if err == nil {
			break
		}
		fmt.Println("Only numerical input is allowed.\n")
	}
	return index
}

func sendBalance(seed string, addrs []string, balances []uint64, i int, target string) {

	transfers := bundle.Transfers{
		{
			Address: target,
			Value:   balances[i],
		},
	}
	inputs := []api.Input{
		{

			Address:  addrs[i],
			Security: consts.SecurityLevelMedium,
			KeyIndex: uint64(i),
			Balance:  balances[i],
		},
	}
	prepTransferOpts := api.PrepareTransfersOptions{Inputs: inputs}

	// prepare the transfer by creating a bundle with the given transfers and inputs.
	// the result are trytes ready for PoW.
	trytes, err := iotaAPI.PrepareTransfers(seed, transfers, prepTransferOpts)
	if err != nil {
		// handle error
		panic(err)
	}

	fmt.Println(trytes)
}
