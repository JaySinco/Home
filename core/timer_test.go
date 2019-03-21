package core

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func ExampleTimerEngine() {
	NewTimerEngine().Run(NewPriceEvent(time.Now()))
}

func NewPriceEvent(deadline time.Time) Event {
	return &nobleMetal{deadline, nil}
}

type Price struct {
	timestamp int64
	goldBuy   float32
	goldSell  float32
}

func (p Price) String() string {
	tm := time.Unix(p.timestamp, 0)
	return fmt.Sprintf("%s#gold{%.2f,%.2f}",
		tm.Format("15:04:05"), p.goldBuy, p.goldSell)
}

const nobleMetalUrl = "http://www.icbc.com.cn/ICBCDynamicSite/Charts/GoldTendencyPicture.aspx"

var goldRegxp = regexp.MustCompile(`人民币账户黄金(?s:.)*?(\d\d\d\.\d\d)(?s:.)*?(\d\d\d\.\d\d)`)

type nobleMetal struct {
	deadline time.Time
	data     *Price
}

func (n *nobleMetal) Deadline() time.Time {
	return n.deadline
}

func (n *nobleMetal) Trigger() (chain []Event, err error) {
	start := time.Now()
	tick, err := time.ParseDuration(Config().Metal.PriceTick)
	if err != nil {
		return nil, err
	}
	defer func() { chain = append(chain, &nobleMetal{start.Add(tick), nil}) }()
	resp, err := http.Get(nobleMetalUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	goldPrice := goldRegxp.FindSubmatch(body)
	if len(goldPrice) != 3 {
		return nil, fmt.Errorf("failed to find gold price: %s", string(body))
	}
	goldBuy, _ := strconv.ParseFloat(string(goldPrice[1]), 32)
	goldSell, _ := strconv.ParseFloat(string(goldPrice[2]), 32)
	n.data = new(Price)
	n.data.timestamp = time.Now().Unix()
	n.data.goldBuy, n.data.goldSell = float32(goldBuy), float32(goldSell)
	return
}

func (n *nobleMetal) String() string {
	return fmt.Sprintf("%s@%s", "Metal_Price", n.deadline.Format("15:04:05"))
}
