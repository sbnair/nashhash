package gm

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// DisconnectOperator command to stop a game
	DisconnectOperator = 0
	// EthClientPath is the path to the geth.ipc
	EthClientPath = "/Users/me/Library/Ethereum/rinkeby/geth.ipc"
)

// GameOperator operates a game contract
type GameOperator struct {
	contractAddress string
	controlChannel  chan int
	playing         bool

	// GM controlling us
	gm *GM
}

// ContractAddress returns the address of the game contract
// the handler is operating
func (gop *GameOperator) ContractAddress() string {
	return gop.contractAddress
}

// Playing returns true of the operator is playing a game
// right now
func (gop *GameOperator) Playing() bool {
	return gop.playing
}

// Init the handler
func (gop *GameOperator) Init(addr string, gm *GM) {
	gop.contractAddress = addr
	gop.controlChannel = make(chan int)
	gop.playing = false
	gop.gm = gm
}

// Play the game at the contract address
func (gop *GameOperator) Play() error {
	if gop.playing == true {
		log.Printf("WARNING GameOperator: game %s already operated\n", gop.contractAddress)
		return errors.New("GameOperator: game operated")
	}
	go gop.playGame()
	gop.playing = true
	return nil
}

// Game control logic goes into this function
func (gop *GameOperator) playGame() {
	log.Printf("INFO GameOperator %s: playing the game\n", gop.contractAddress)
	for {

		// We use select here for non-blocking read on the socket
		select {
		case cmd := <-gop.controlChannel:

			// Different commands can be passed here.
			// For example, we can ask the handler to reset the state
			// or log the current state of the game
			switch cmd {
			case DisconnectOperator:
				log.Printf("INFO GameOperator %s: quitting the game\n", gop.contractAddress)
				//fmt.Printf("operator quitting the game at address %s\n", gop.contractAddress)
				return
			default:
				log.Printf("WARNING GameOperator %s: unknown command\n", gop.contractAddress)
				//fmt.Printf("Unknown command")
			}

		// The default behaviour is to continue operating the game
		default:
			// Proceed with the normal game logic
			if gop.gm.debug == true {
				log.Printf("INFO GameOperator %s: operate succesful\n", gop.contractAddress)
			} else {
				gop.operate()
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// This function is the heart of operator. Does the following:
// 1) Figure out in which state the game is
// 2) Force a state transition or payout

const (
	GameCommitState = 0
	GameRevealState = 1
	GamePayoutState = 2
)

func (gop *GameOperator) operate() {

	// Create an IPC based RPC connection to a remote node
	conn, err := ethclient.Dial(EthClientPath)
	if err != nil {
		log.Printf("Failed to connect to the Ethereum client: %v", err)
	}
	// Instantiate the contract and display its name
	game, err := NewGame(common.HexToAddress(gop.contractAddress), conn)
	if err != nil {
		log.Printf("Failed to instantiate a Game contract: %v", err)
	}

	auth := gop.gm.auth

	state, err := game.GetGameState(nil)
	if err != nil {
		log.Printf("Failed to retrieve game state: %v", err)
	}

	//Initiate appropriate transitions
	switch s := state.Int64(); s {
	case GameCommitState:
		tx, txerr := game.ForceToRevealState(auth)
		if txerr != nil {
			log.Printf("Failed to force game into reveal: %v", txerr)
		} else {
			log.Printf("INFO game operator: ForceToReveal SUCCESS 0x%x\n", tx.Hash())
		}

	case GameRevealState:
		tx, txerr := game.ForceToPayoutState(auth)
		if txerr != nil {
			log.Printf("Failed to force game into payout: %v", txerr)
		} else {
			log.Printf("INFO game operator: ForceToPayout SUCCESS 0x%x\n", tx.Hash())
		}

	case GamePayoutState:
		tx, txerr := game.ResetGame(auth)
		if txerr != nil {
			log.Printf("Failed to perform payout: %v", txerr)
		} else {
			log.Printf("INFO game operator: SUCCSFUL RESET 0x%x\n", tx.Hash())
		}

	default:
		log.Println("game operator: unknown operation state")

	}

	fmt.Println("Game State:", state)

	//fmt.Printf("wow look at me I am operating hard %s\n", gop.contractAddress)
	log.Printf("INFO GameOperator %s: operate succesful\n", gop.contractAddress)
}

// Stop stops the operator from operating the game. The game ideally should also
// be paused.
func (gop *GameOperator) Stop() error {
	if gop.playing == false {
		log.Printf("WARNING GameOperator %s: game already stopped\n", gop.contractAddress)
		return errors.New("game already stopped")
	}
	gop.controlChannel <- DisconnectOperator
	gop.playing = false
	return nil
}
