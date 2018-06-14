//go:generate abigen --sol ./../../contracts/Game.sol --pkg gm --out Game.go
package gm

import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// The GM object. Runs on the server. Manages GameOperators, connects
// new games, resets games and etc etc etc. Important mister.
type GM struct {
	// handlers      []GameOperator
	operatedGames map[string]*GameOperator
	gmLock        sync.Mutex

	// Ethereum stuff
	auth *bind.TransactOpts

	// debug mode
	debug bool

	// RPC stuff
	dead bool
	l    net.Listener
	port int
}

// Init initializes the game master. In particular, it should register the
// game master for RPC.
func (gm *GM) Init(ipAddr string, port int, hexkey string, debug bool) error {

	gm.gmLock.Lock()
	defer gm.gmLock.Unlock()

	gm.debug = debug

	if debug == false {

		// We need to create a transactor to be able to execute
		// contract functions
		privk, err := ethcrypto.HexToECDSA(hexkey)
		if err != nil {
			log.Fatalf("GM: bad private key")
		}

		gm.auth = bind.NewKeyedTransactor(privk)
		if gm.auth == nil {
			log.Fatalf("GM: failed to create authorized transactor: %v", err)
		}
	}

	gm.operatedGames = make(map[string]*GameOperator)

	// RPC RELATED STUFF BELOW
	// Register our baby with net/rpc
	gm.port = port

	rpcs := rpc.NewServer()
	rpcs.Register(gm)

	// Create a TCP listener that will listen on `Port`
	l, e := net.Listen("tcp", ipAddr+":"+strconv.Itoa(gm.port))
	if e != nil {
		log.Fatal("listen error: ", e)
	}
	gm.l = l

	// Go routine that accepts and serves new procedure calls
	go func() {
		for gm.dead == false {
			conn, err := gm.l.Accept()
			if err == nil && gm.dead == false {
				go rpcs.ServeConn(conn)
			} else if err == nil {
				conn.Close()
			}
			if err != nil && gm.dead == false {
				log.Printf("ERROR GM accept: %v\n", err.Error())
				gm.Kill()
			}
		}
	}()
	log.Printf("INFO GM: Initialization succesful.\n")

	return nil
}

// Execute is a test
func (gm *GM) Execute(req ExecuteCallArgs, res *ExecuteCallReply) error {
	if req.Message == "" {
		return errors.New("You must give me a message")
	}

	res.Response = "This is your message: " + req.Message
	return nil
}

// Connect call connects a GameOperator to a game at ConnectCallArgs.ContractAddress
func (gm *GM) Connect(args ConnectCallArgs, res *ConnectCallReply) error {
	gm.gmLock.Lock()
	defer gm.gmLock.Unlock()

	// First we check if the game already has an operator on it
	addr := args.ContractAddress
	if gm.isOperated(addr) {
		log.Printf("WARNING GM: %s already operated\n", addr)
		return errors.New("GM: game already operated")
	}

	// Create a game operator
	gop := &GameOperator{}
	gop.Init(addr, gm)

	// Add this GameOperator to the mapping
	gm.operatedGames[addr] = gop

	// TODO: Give an option to not immediately start operating the game
	e := gm.operatedGames[addr].Play()
	if e != nil {
		log.Fatalf("ERROR GM: inconsistent state")
		panic("Error: inconsistent state in GM.Connect()")
		//		return e
	}

	log.Printf("INFO GM: %s connected succesfully\n", addr)

	return nil
}

// Disconnect call disconnects a GameOperator from a game at ConnectCallArgs.ContractAddress
func (gm *GM) Disconnect(args DisconnectCallArgs, res *DisconnectCallReply) error {
	gm.gmLock.Lock()
	defer gm.gmLock.Unlock()

	// First we check if the game already has an operator on it
	addr := args.ContractAddress
	if !gm.isOperated(addr) {
		return errors.New("GM: game at " + addr + " not operated")
	}

	// Stop the handler
	e := gm.operatedGames[addr].Stop()
	if e != nil {
		panic("Error: inconsistent state in GM.Connect()")
		//		return e
	}

	// Remove the gameOperator from the map
	delete(gm.operatedGames, addr)

	log.Printf("INFO GM: %s disconnected succesfully\n", addr)

	return nil
}

// Helper, checks if a game is already operated
func (gm *GM) isOperated(addr string) bool {
	if _, ok := gm.operatedGames[addr]; ok {
		return true
	}
	return false
}

// Kill the GM is something is wrong.
func (gm *GM) Kill() {

	gm.gmLock.Lock()
	defer gm.gmLock.Unlock()

	for _, v := range gm.operatedGames {
		v.Stop()
	}

	log.Printf("INFO GM: all game operators stopped\n")

	gm.dead = true
	gm.l.Close()

	log.Printf("INFO GM: dead\n")
}