package db

import (
	_ "embed"
)

//go:embed historical_seats_seed.sql
var historicalSeatsSeedSQL string
