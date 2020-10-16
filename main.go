/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"time"

	ontology_go_sdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology/common"
	"github.com/siovanus/reserve-snapshot/config"
	"github.com/siovanus/reserve-snapshot/log"
	"github.com/urfave/cli"
)

func setupApp() *cli.App {
	app := cli.NewApp()
	app.Usage = "snapshot reserve server"
	app.Action = startServer
	app.Copyright = "Copyright in 2018 The Ontology Authors"
	app.Flags = []cli.Flag{
		config.LogLevelFlag,
		config.ConfigPathFlag,
	}
	app.Before = func(context *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}
	return app
}

func main() {
	if err := setupApp().Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func startServer(ctx *cli.Context) {
	logLevel := ctx.GlobalInt(config.GetFlagName(config.LogLevelFlag))
	log.InitLog(logLevel, log.PATH, log.Stdout)

	var err error
	configPath := ctx.GlobalString(config.GetFlagName(config.ConfigPathFlag))
	config.DefConfig, err = config.NewConfig(configPath)
	if err != nil {
		log.Errorf("parse config failed, err: %s", err)
		return
	}
	sdk := ontology_go_sdk.NewOntologySdk()
	sdk.ClientMgr.NewRpcClient().SetAddress(config.DefConfig.JsonRpcAddress)
	flashAddress, err := common.AddressFromHexString(config.DefConfig.FlashPoolAddress)
	if err != nil {
		log.Errorf("common.AddressFromHexString, err: %s", err)
		return
	}

	now := time.Now().UTC()
	next := time.Date(2020, time.October, 22, 23, 59, 0, 0, time.UTC)
	t := time.NewTimer(next.Sub(now))
	<-t.C
	log.Infof("snapshot start: %v", time.Now().UTC().String())
	for {
		// 以下为定时执行的操作
		now := time.Now().UTC()
		f, err := os.Create(fmt.Sprintf("data/%s", now.String()))
		w := bufio.NewWriter(f)
		result, err := TotalReserve(sdk, flashAddress)
		if err != nil {
			log.Errorf("TotalReserve, err: %s", err)
		}
		for k, v := range result {
			w.WriteString(k)
			w.WriteString("\t")
			w.WriteString(v)
			w.WriteString("\n")
		}
		w.Flush()
		f.Close()
		if now.After(next.Add(2 * time.Minute)) {
			log.Infof("Done")
			return
		}
	}
}

func GetAllMarkets(sdk *ontology_go_sdk.OntologySdk, flashAddress common.Address) ([]common.Address, error) {
	preExecResult, err := sdk.WasmVM.PreExecInvokeWasmVMContract(flashAddress,
		"allMarkets", []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("getAllMarkets, this.sdk.WasmVM.PreExecInvokeWasmVMContract error: %s", err)
	}
	r, err := preExecResult.Result.ToByteArray()
	if err != nil {
		return nil, fmt.Errorf("getAllMarkets, preExecResult.Result.ToByteArray error: %s", err)
	}
	source := common.NewZeroCopySource(r)
	allMarkets := make([]common.Address, 0)
	l, _, irregular, eof := source.NextVarUint()
	if irregular || eof {
		return nil, fmt.Errorf("getAllMarkets, source.NextVarUint error")
	}
	for i := 0; uint64(i) < l; i++ {
		addr, eof := source.NextAddress()
		if eof {
			return nil, fmt.Errorf("getAllMarkets, source.NextAddress error")
		}
		allMarkets = append(allMarkets, addr)
	}
	return allMarkets, nil
}

func TotalReserve(sdk *ontology_go_sdk.OntologySdk, flashAddress common.Address) (map[string]string, error) {
	allMarkets, err := GetAllMarkets(sdk, flashAddress)
	if err != nil {
		return nil, fmt.Errorf("TotalReserve, this.GetAllMarkets error: %s", err)
	}
	result := make(map[string]string)
	for _, address := range allMarkets {
		name := config.DefConfig.AssetMap[address.ToHexString()]
		reserveBalance, err := getTotalReserves(sdk, address)
		if err != nil {
			return nil, fmt.Errorf("TotalReserve, this.getTotalReserves error: %s", err)
		}
		result[name] = reserveBalance.String()
	}
	return result, nil
}

func getTotalReserves(sdk *ontology_go_sdk.OntologySdk, contractAddress common.Address) (*big.Int, error) {
	preExecResult, err := sdk.WasmVM.PreExecInvokeWasmVMContract(contractAddress,
		"totalReserves", []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("getTotalReserves, this.sdk.WasmVM.PreExecInvokeWasmVMContract error: %s", err)
	}
	r, err := preExecResult.Result.ToByteArray()
	if err != nil {
		return nil, fmt.Errorf("getTotalReserves, preExecResult.Result.ToByteArray error: %s", err)
	}
	source := common.NewZeroCopySource(r)
	amount, eof := source.NextI128()
	if eof {
		return nil, fmt.Errorf("getTotalReserves, source.NextI128 error")
	}
	return amount.ToBigInt(), nil
}
