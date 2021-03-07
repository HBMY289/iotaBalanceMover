package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/bundle"
	"github.com/iotaledger/iota.go/consts"
)

var iotaAPI *api.API
var acc accountState

const defaultNode = "https://nodes.iota.org:443"
const seedLen = 81
const dummySeed = "999999999999999999999999999999999999999999999999999999999999999999999999999999999"
const addrsPerBatch = 500
const depth = 3
const mwm = 14

func main() {
	fmt.Println("\nWelcome!\nThis program will list all addresses of your seed with a positive balance and will let you move the funds of a specific address.")
	getAPI()
	getSeed()

	for {
		getAccountState()
		printAccountState()
		moveBalance()
		fmt.Print("\nDo you want to move funds of another address of this seed? (y/n): ")
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

func getSeed() {
	var seed, answer string
	for {
		fmt.Print("\nEnter seed: ")
		fmt.Scanln(&seed)
		if !hasInvalidChars(seed) {
			if !hasInvalidChars(seed) {
				if len(seed) == seedLen {
					break
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
					break
				}
			}
		} else {
			fmt.Println("\nValid seeds only contain upper case letters A-Z and the number 9.")
		}
	}
	acc.seed = seed
}

func hasInvalidChars(seed string) bool {
	for _, r := range seed {
		if (r < 'A' || r > 'Z') && r != '9' {
			return true
		}
	}
	return false

}

func getAccountState() {

	var answer string
	index := 0
	acc.resetState()
	for {
		fmt.Printf("\nGenerating addresses #%d to #%d\n", index, index+addrsPerBatch)
		addrBatch := generateAddresses(index, addrsPerBatch)
		acc.addrs = append(acc.addrs, addrBatch...)
		balancesBatch := getBalances(addrBatch)
		acc.balances = append(acc.balances, balancesBatch...)
		if acc.totalBalance() > 0 {
			fmt.Printf("\nFound a total of %di on the first %d addresses.\nIs the total balance correct? (y/n):", acc.totalBalance(), index+addrsPerBatch)
			fmt.Scanln(&answer)
			if answer == "y" {
				fmt.Println()
				break
			}
		}
		index += addrsPerBatch
	}
	getSpentStates()
}

func generateAddresses(start, count int) []string {
	var addrs []string
	addrs, err := address.GenerateAddresses(acc.seed, uint64(start), uint64(count), consts.SecurityLevelMedium, true)
	if err != nil {
		panic(err)
	}
	return addrs
}

func getBalances(addrs []string) []uint64 {
	balances, err := iotaAPI.GetBalances(addrs)
	if err != nil {
		panic(err)
	}
	return balances.Balances
}

func getSpentStates() {
	spentStates, err := iotaAPI.WereAddressesSpentFrom(acc.addrs...)
	if err != nil {
		panic(err)
	}
	acc.spent = spentStates
}

func printAccountState() {
	fmt.Println("Listing all addresses with a positive balance:")
	for i, addr := range acc.addrs {
		balance := strconv.Itoa(int(acc.balances[i])) + "i"
		if acc.balances[i] > 0 {
			if acc.spent[i] {
				balance = inRed("(" + balance + ")")
			}
			fmt.Printf("address #%d: %s %s\n", i, balance, addr)
		}
	}
	fmt.Printf("Total balance: %di\n", acc.totalBalance())
	if acc.fundsOnSpentAddr() {
		fmt.Printf("Funds on spent addresses are shown in %s.\n", inRed("( )"))
	}
}

func inRed(text string) string {
	colorRed := "\033[31m"
	colorReset := "\033[0m"
	return string(colorRed) + text + string(colorReset)
}

func moveBalance() {
	var confirm string
	index := getChosenIndex()

	if acc.spent[index] {
		fmt.Printf("%s\nThe chosen address was already used for spending.\nSending multiple times from the same address can put these funds at risk.\nAre you sure you want to proceed? (y/n):", inRed("WARNING!!!"))
		fmt.Scanln(&confirm)
		if confirm != "y" {
			return
		}
	}
	target := getTargetAddress()
	fmt.Printf("\n\nMoving %di from address \n%s\nto address\n%s\nDo you want to proceed? (y/n):", acc.balances[index], acc.addrs[index], target)

	fmt.Scanln(&confirm)
	if confirm == "y" {
		fmt.Println("\nSending transaction")
		sendBalance(index, target)
	}

}

func getChosenIndex() int {
	var index int
	for {
		var input string
		fmt.Print("\nYou can move the funds of an address by entering its index number.\nAddress index:")
		fmt.Scanln(&input)
		var err error
		index, err = strconv.Atoi(input)
		if err != nil {
			fmt.Println("Only numerical input is allowed.")
		} else {
			if acc.balances[index] > 0 {
				break
			}
			fmt.Printf("Address #%d does not have a balance.\n", index)
		}

	}
	return index
}

func getTargetAddress() string {
	var addr string
	for {
		fmt.Print("Enter the address you want to move the funds to: ")
		fmt.Scanln(&addr)
		if address.ValidAddress(addr) == nil {
			break
		}
		fmt.Println("Please enter valid iota address with checksum (90 characters).")
	}
	return addr
}
func sendBalance(i int, target string) {

	transfers := bundle.Transfers{
		{
			Address: target,
			Value:   acc.balances[i],
		},
	}
	inputs := []api.Input{
		{

			Address:  acc.addrs[i],
			Security: consts.SecurityLevelMedium,
			KeyIndex: uint64(i),
			Balance:  acc.balances[i],
		},
	}
	prepTransferOpts := api.PrepareTransfersOptions{Inputs: inputs}

	trytes, err := iotaAPI.PrepareTransfers(acc.seed, transfers, prepTransferOpts)
	if err != nil {
		panic(err)
	}
	b, err := iotaAPI.SendTrytes(trytes, depth, mwm)
	if err != nil {
		panic(err)
	}
	hash := bundle.TailTransactionHash(b)
	fmt.Printf("\nSuccessfully sent transaction:\n%s\n", hash)
	confirmTx(hash)
}

func confirmTx(orgHash string) {
	hash := orgHash
	fmt.Println("\nStart confirming transaction")
	for !confirmed(hash) {
		ok, err := iotaAPI.IsPromotable(hash)
		if err != nil {
			fmt.Println(err)
		} else {
			if ok {
				spamTransfers := bundle.Transfers{bundle.EmptyTransfer}
				opts := api.SendTransfersOptions{Reference: &hash}
				_, err := iotaAPI.SendTransfer(dummySeed, depth, mwm, spamTransfers, &opts)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println("Promoted transaction")
				}
			} else {
				b, err := iotaAPI.ReplayBundle(hash, depth, mwm)
				if err != nil {
					fmt.Println(err)
				} else {
					hash = b[0].Hash
					fmt.Printf("Reattached transaction. New hash: %s\n", hash)
				}
			}
		}
		time.Sleep(time.Second)
	}
	fmt.Println("\nTransaction is confirmed.")
}
func confirmed(hash string) bool {
	states, err := iotaAPI.GetInclusionStates([]string{hash})
	if err != nil {
		fmt.Println(err)
		return false
	}
	if len(states) > 0 && states[0] {
		return true
	}
	return false
}

func (s *accountState) resetState() {
	s.addrs = nil
	s.balances = nil
	s.spent = nil
}

func (s *accountState) fundsOnSpentAddr() bool {
	for i, s := range s.spent {
		if s && acc.balances[i] > 0 {
			return true
		}
	}
	return false
}

func (s *accountState) totalBalance() uint64 {
	var sum uint64
	for _, b := range s.balances {
		sum += b
	}
	return sum
}

type accountState struct {
	seed     string
	addrs    []string
	balances []uint64
	spent    []bool
}
