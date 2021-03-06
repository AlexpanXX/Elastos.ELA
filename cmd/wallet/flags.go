package wallet

import (
	"github.com/elastos/Elastos.ELA/account"

	"github.com/urfave/cli"
)

var (
	// Account flags
	AccountWalletFlag = cli.StringFlag{
		Name:  "wallet, w",
		Usage: "wallet `<file>` path, ",
		Value: account.KeystoreFileName,
	}
	AccountPasswordFlag = cli.StringFlag{
		Name:  "password, p",
		Usage: "wallet password",
	}
	AccountMultiMFlag = cli.IntFlag{
		Name:  "m",
		Usage: "Min signature `<number>` of multi signature address",
	}
	AccountMultiPubKeyFlag = cli.StringFlag{
		Name:  "pubkeys, pks",
		Usage: "Public key list of multi signature address, separate public keys with comma `,`",
	}

	// Transaction flags
	TransactionFromFlag = cli.StringFlag{
		Name:  "from",
		Usage: "the spend `<address>` of the transaction",
	}
	TransactionToFlag = cli.StringFlag{
		Name:  "to",
		Usage: "the receive `<address>` of the transaction",
	}
	TransactionAmountFlag = cli.StringFlag{
		Name:  "amount",
		Usage: "the transfer `<amount>` of the transaction",
	}
	TransactionFeeFlag = cli.StringFlag{
		Name:  "fee",
		Usage: "the transfer `<fee>` of the transaction",
	}
	TransactionLockFlag = cli.StringFlag{
		Name:  "lock",
		Usage: "the `<lock time>` to specify when the received asset can be spent",
	}
	TransactionHexFlag = cli.StringFlag{
		Name:  "hex",
		Usage: "the transaction content in hex string format to be sign or send",
	}
	TransactionFileFlag = cli.StringFlag{
		Name:  "file, f",
		Usage: "the file path to specify a transaction file path with the hex string content to be sign",
	}
)
