package main

import (
	"flag"
	"fmt"
	"ignite/models"
	"ignite/ss"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/go-xorm/xorm"
	toml "github.com/pelletier/go-toml"
)

var (
	conf = flag.String("c", "./config.toml", "config file")
	db   *xorm.Engine
)

const (
	GB = 1024 * 1024 * 1024
)

func init() {

	// Load config file
	if _, err := os.Stat(*conf); os.IsNotExist(err) {
		log.Println("Cannot load config.toml, file doesn't exist...")
		os.Exit(1)
	}

	config, err := toml.LoadFile(*conf)

	if err != nil {
		log.Println("Failed to load config file:", *conf)
		os.Exit(1)
	}

	//Init DB connection
	var (
		user     = config.Get("mysql.user").(string)
		password = config.Get("mysql.password").(string)
		host     = config.Get("mysql.host").(string)
		dbname   = config.Get("mysql.dbname").(string)
	)
	connString := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", user, password, host, dbname)
	db, err = xorm.NewEngine("mysql", connString)
	if err != nil {
		log.Println("Create mysql engine error:", err.Error())
		os.Exit(1)
	}

	err = db.Ping()

	if err != nil {
		log.Println("Cannot connetc to database:", err.Error())
		os.Exit(1)
	}
}

func main() {
	log.Println("Start ...")
	// 1. Load all service from user
	users := []models.User{}
	err := db.Where("service_id != '' AND status != 2").Find(&users)
	if err != nil {
		log.Println("Get users error: ", err.Error())
		os.Exit(1)
	}

	// 2. Compute ss bandwidth
	for _, user := range users {
		raw, err := ss.GetContainerStatsOutNet(user.ServiceId)
		if err != nil {
			log.Printf("Get container(%s) net out error: %s\n", user.ServiceId, err.Error())
			continue
		}

		// Get container start time
		startTime, err := ss.GetContainerStartTime(user.ServiceId)
		if err != nil {
			log.Printf("Get container(%s) start time error: %s\n", user.ServiceId, err.Error())
			continue
		}

		// Update user package used
		var bandwidth float32
		if user.LastStatsTime == nil || user.LastStatsTime.Before(*startTime) {
			bandwidth = float32(float64(raw) / GB)
		} else {
			bandwidth = float32(float64(raw-user.LastStatsResult) / GB)
		}
		user.PackageUsed += bandwidth

		if int(user.PackageUsed) >= user.PackageLimit {
			// Stop container && update user status
			err := ss.StopContainer(user.ServiceId)
			if err != nil {
				log.Println("Stop container(%s) error: %s\n", user.ServiceId, err.Error())
			} else {
				log.Printf("STOP: user(%d-%s)-container(%s)\n", user.Id, user.Username, user.ServiceId[:12])
				user.Status = 2
			}
		}

		// 3. Update user stats info
		now := time.Now()
		user.LastStatsTime = &now
		user.LastStatsResult = raw
		_, err = db.Id(user.Id).Cols("package_used", "last_stats_result", "last_stats_time", "status").Update(user)
		if err != nil {
			log.Printf("Update user(%d) error: %s\n", user.Id, err.Error())
			continue
		}
		log.Printf("STATS: user(%d-%s)-container(%s)-bandwidth(%.2f)\n", user.Id, user.Username, user.ServiceId[:12], bandwidth)
	}
	log.Println("Done !")
}
