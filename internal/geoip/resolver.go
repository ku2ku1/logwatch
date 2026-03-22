package geoip

import (
	"log"
	"net"

	"github.com/oschwald/geoip2-golang"
)

type Location struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lon"`
}

type Resolver struct {
	db *geoip2.Reader
}

func New(path string) (*Resolver, error) {
	db, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}
	log.Printf("[geoip] database loaded: %s", path)
	return &Resolver{db: db}, nil
}

func (r *Resolver) Lookup(ipStr string) *Location {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil
	}
	record, err := r.db.City(ip)
	if err != nil {
		return nil
	}
	country := record.Country.Names["en"]
	city := record.City.Names["en"]
	if country == "" {
		return nil
	}
	return &Location{
		Country:     country,
		CountryCode: record.Country.IsoCode,
		City:        city,
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
	}
}

func (r *Resolver) Close() {
	if r.db != nil {
		r.db.Close()
	}
}
