package main

import (
	"net/http"
	"log"
)

func main() {
	// 设置静态文件服务器
	fs := http.FileServer(http.Dir("."))
	
	// 添加CORS支持
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		fs.ServeHTTP(w, r)
	})
	
	log.Println("🌐 前端演示服务启动在: http://localhost:3001")
	log.Println("📱 访问演示页面: http://localhost:3001/demo.html")
	log.Fatal(http.ListenAndServe(":3001", handler))
}