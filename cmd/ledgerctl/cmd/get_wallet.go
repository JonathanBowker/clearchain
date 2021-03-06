package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tendermint/clearchain/client"
	"github.com/tendermint/go-wire"
)

func init() {
	RootCmd.AddCommand(&cobra.Command{
		Use:   "get_wallet",
		Short: "get wallet for an account from blockchain",
		Run: func(cmd *cobra.Command, args []string) {

			var chainID, serverAddress, accountID string

			if len(args) == 4 {
				//TMSP version
				//ledgerctl get_wallet test_chain_id tcp://127.0.0.1:46658 ATRXWwlJ6bvNRcNRT/EMmymjZvAGsLZp5a95t9HL5NRhhDh4uTLuSQikLSS//AOeuN+s1DQMgzQjEGgglAR/r6s= 1d2df1ae-accb-11e6-bbbb-00ff5244ae7f
				//Websocket version
				//ledgerctl get_wallet test_chain_id 127.0.0.1:46657 ATRXWwlJ6bvNRcNRT/EMmymjZvAGsLZp5a95t9HL5NRhhDh4uTLuSQikLSS//AOeuN+s1DQMgzQjEGgglAR/r6s= 1d2df1ae-accb-11e6-bbbb-00ff5244ae7f

				chainID = args[0]
				serverAddress = args[1]
				accountID = args[2]
			} else {
				chainID = readParameter("chainID")
				serverAddress = readParameter("serverAddress")
				accountID = readParameter("accountID")
			}
			
			client.SetChainID(chainID)
			client.StartClient(serverAddress)
			fmt.Println(string(wire.JSONBytes(client.GetAccount(accountID).Account[0])))
		},
	})
}
