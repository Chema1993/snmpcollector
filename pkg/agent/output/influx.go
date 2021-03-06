package output

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/toni-moreno/snmpcollector/pkg/config"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	log *logrus.Logger
)

// SetLogger adds a logger to this module
func SetLogger(l *logrus.Logger) {
	log = l
}

/*InfluxDB database export */
type InfluxDB struct {
	Sent        int64
	Errors      int64
	cfg         *config.InfluxCfg
	initialized bool
	imutex      sync.Mutex
	started     bool
	smutex      sync.Mutex

	dummy  bool
	iChan  chan *client.BatchPoints
	chExit chan bool
	client client.Client
}

var DummyDB = &InfluxDB{
	cfg:         nil,
	initialized: false,
	started:     false,
	dummy:       true,
	iChan:       nil,
	chExit:      nil,
	client:      nil,
	Sent:        0,
	Errors:      0,
}

func (db *InfluxDB) incSent() {
	atomic.AddInt64(&db.Sent, 1)
}

func (db *InfluxDB) addSent(n int64) {
	atomic.AddInt64(&db.Sent, n)
}

func (db *InfluxDB) incErrors() {
	atomic.AddInt64(&db.Errors, 1)
}

func (db *InfluxDB) addErrors(n int64) {
	atomic.AddInt64(&db.Errors, n)
}

//BP create a Batch point influx object
func (db *InfluxDB) BP() (*client.BatchPoints, error) {
	if db.dummy == true {
		bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
			Database:        "dbdummy",
			RetentionPolicy: "dbretention",
			Precision:       "ns", //Default precision for Time lib
		})
		return &bp, nil
	}
	//
	if len(db.cfg.Retention) == 0 {
		db.cfg.Retention = "autogen"
	}
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        db.cfg.DB,
		RetentionPolicy: db.cfg.Retention,
		Precision:       db.cfg.Precision, //Default precision for Time lib
	})
	if err != nil {
		log.Errorf("Error on create BatchPoint: %s", err)
		return nil, err
	}
	return &bp, err
}

// Ping InfluxDB Server
func Ping(cfg *config.InfluxCfg) (client.Client, time.Duration, string, error) {

	conf := client.HTTPConfig{
		Addr:      fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port),
		Username:  cfg.User,
		Password:  cfg.Password,
		UserAgent: cfg.UserAgent,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
	}
	cli, err := client.NewHTTPClient(conf)

	if err != nil {
		return cli, 0, "", err
	}
	elapsed, message, err := cli.Ping(time.Duration(cfg.Timeout) * time.Second)
	return cli, elapsed, message, err
}

//Connect to influxdb
func (db *InfluxDB) Connect() error {
	if db.dummy == true {
		return nil
	}
	var err error
	db.client, _, _, err = Ping(db.cfg)
	return err
}

// CheckAndSetStarted check if this thread is already working and set if not
func (db *InfluxDB) CheckAndSetStarted() bool {
	db.smutex.Lock()
	defer db.smutex.Unlock()
	retval := db.started
	db.started = true
	return retval
}

// CheckAndUnSetStarted check if this thread is already working and unset if not
func (db *InfluxDB) CheckAndUnSetStarted() bool {
	db.smutex.Lock()
	defer db.smutex.Unlock()
	retval := db.started
	db.started = false
	return retval
}

// IsStarted check if this thread is already working
func (db *InfluxDB) IsStarted() bool {
	db.smutex.Lock()
	defer db.smutex.Unlock()
	return db.started
}

// SetStartedAs change started state
func (db *InfluxDB) SetStartedAs(st bool) {
	db.smutex.Lock()
	defer db.smutex.Unlock()
	db.started = st
}

// CheckAndSetInitialized check if this thread is already working and set if not
func (db *InfluxDB) CheckAndSetInitialized() bool {
	db.imutex.Lock()
	defer db.imutex.Unlock()
	retval := db.initialized
	db.initialized = true
	return retval
}

// CheckAndUnSetInitialized check if this thread is already working and set if not
func (db *InfluxDB) CheckAndUnSetInitialized() bool {
	db.imutex.Lock()
	defer db.imutex.Unlock()
	retval := db.initialized
	db.initialized = false
	return retval
}

// NewNotInitInfluxDB Create Object in memory but not initialized until ready connection needed
func NewNotInitInfluxDB(c *config.InfluxCfg) *InfluxDB {
	return &InfluxDB{
		cfg:     c,
		dummy:   false,
		started: false,
		Sent:    0,
		Errors:  0,
	}
}

//Init initialies runtime info
func (db *InfluxDB) Init() {
	if db.dummy == true {
		return
	}

	if db.CheckAndSetInitialized() == true {
		log.Infof("Sender thread to : %s  already Initialized (skipping Initialization)", db.cfg.ID)
		return
	}

	if len(db.cfg.UserAgent) == 0 {
		db.cfg.UserAgent = "snmpCollector-" + db.cfg.ID
	}

	log.Infof("Initializing influxdb with id = [ %s ]", db.cfg.ID)

	log.Infof("Connecting to: %s", db.cfg.Host)
	db.iChan = make(chan *client.BatchPoints, 65535)
	db.chExit = make(chan bool)
	if err := db.Connect(); err != nil {
		log.Errorln("failed connecting to: ", db.cfg.Host)
		log.Errorln("error: ", err)
		//if no connection done started = false and it will try to test again later??
		return
	}

	log.Infof("Connected to: %s", db.cfg.Host)
}

func (db *InfluxDB) End() {
	if db.dummy == true {
		return
	}
	if db.CheckAndUnSetInitialized() == true {
		close(db.iChan)
		close(db.chExit)
		db.client.Close()
	}
}

// End finalize sender goroutines
func (db *InfluxDB) StopSender() {
	if db.dummy == true {
		return
	}

	if db.IsStarted() == true {
		db.chExit <- true
		return
	}

	log.Infof("Can not stop Sender [%s] becaouse of it is already stopped", db.cfg.ID)
}

//Send send data
func (db *InfluxDB) Send(bps *client.BatchPoints) {
	if db.dummy == true {
		return
	}
	db.iChan <- bps
}

//Hostname get hostname
func (db *InfluxDB) Hostname() string {
	return strings.Split(db.cfg.Host, ":")[0]
}

// StartSender begins sender loop
func (db *InfluxDB) StartSender(wg *sync.WaitGroup) {
	if db.dummy == true {
		return
	}
	if db.CheckAndSetStarted() == true {
		log.Infof("Sender thread to : %s  already started (skipping Goroutine creation)", db.cfg.ID)
		return
	}
	wg.Add(1)
	go db.startSenderGo(rand.Int(), wg)
}

func (db *InfluxDB) startSenderGo(r int, wg *sync.WaitGroup) {
	defer wg.Done()

	time.Sleep(5)

	log.Infof("beginning Influx Sender thread: [%s]", db.cfg.ID)
	for {
		select {
		case <-db.chExit:
			log.Infof("EXIT from Influx sender process for device [%s] ", db.cfg.ID)
			db.SetStartedAs(false)
			return
		case data := <-db.iChan:
			if data == nil {
				log.Warn("null influx input")
				continue
			}

			for {
				// keep trying until we get it (don't drop the data)
				log.Debugf("sending data from Sender [ %s ] (%d)", db.cfg.ID, r)
				if err := db.client.Write(*data); err != nil {
					db.incErrors()
					log.Errorln("influxdb write error: ", err)
					// try again in a bit
					// TODO: this could be better
					// Todo add InfluxResend on error.
					time.Sleep(30 * time.Second)
					continue
				} else {
					db.incSent()
					break
				}
			}
		}
	}
}
