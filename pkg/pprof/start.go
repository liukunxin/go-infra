package pprof

import (
	"log"
	"net/http"
	"net/http/pprof"
)

func Start() {
	// 启动 pprof 的独立服务器
	go func() {
		pprofMux := http.NewServeMux()
		pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
		pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		log.Println("启动 pprof 服务器，监听端口: 6060")
		if err := http.ListenAndServe(":6060", pprofMux); err != nil {
			log.Fatalf("pprof 服务器启动失败: %v", err)
		}
	}()
}
