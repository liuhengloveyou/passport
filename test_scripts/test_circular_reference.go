package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
)

func main() {
	// 连接数据库
	dbPool, err := pgxpool.New(context.Background(), "host=localhost user=pcdn password=pcdn12321 dbname=pcdn port=5432 sslmode=disable TimeZone=Asia/Shanghai")
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}
	defer dbPool.Close()

	// 设置日志
	err = common.InitLog("/tmp", "debug")
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 开始事务
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		log.Fatalf("无法开始事务: %v", err)
	}
	defer tx.Rollback(context.Background())

	// 测试循环引用检测
	fmt.Println("检测循环引用...")
	circularNodes, err := dao.DetectCircularReference(tx)
	if err != nil {
		log.Fatalf("检测循环引用失败: %v", err)
	}

	if len(circularNodes) > 0 {
		fmt.Printf("发现循环引用，涉及节点: %v\n", circularNodes)
	} else {
		fmt.Println("未发现循环引用")
	}

	// 回滚事务（只是测试，不需要提交）
	tx.Rollback(context.Background())
}