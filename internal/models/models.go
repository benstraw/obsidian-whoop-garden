package models

// UserProfile represents WHOOP user profile data.
type UserProfile struct {
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// BodyMeasurements represents WHOOP body measurement data.
type BodyMeasurements struct {
	HeightMeter    float64 `json:"height_meter"`
	WeightKilogram float64 `json:"weight_kilogram"`
	MaxHeartRate   int     `json:"max_heart_rate"`
}

// CycleScore holds scoring data for a physiological cycle.
type CycleScore struct {
	Strain            float64 `json:"strain"`
	Kilojoule         float64 `json:"kilojoule"`
	AverageHeartRate  int     `json:"average_heart_rate"`
	MaxHeartRate      int     `json:"max_heart_rate"`
}

// Cycle represents a WHOOP physiological cycle (day).
type Cycle struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
	Start          string     `json:"start"`
	End            string     `json:"end"`
	TimezoneOffset string     `json:"timezone_offset"`
	ScoreState     string     `json:"score_state"`
	Score          CycleScore `json:"score"`
}

// RecoveryScore holds recovery scoring data.
type RecoveryScore struct {
	UserCalibrating     bool    `json:"user_calibrating"`
	RecoveryScore       float64 `json:"recovery_score"`
	RestingHeartRate    float64 `json:"resting_heart_rate"`
	HrvRmssdMilli       float64 `json:"hrv_rmssd_milli"`
	Spo2Percentage      float64 `json:"spo2_percentage"`
	SkinTempCelsius     float64 `json:"skin_temp_celsius"`
}

// Recovery represents WHOOP recovery data linked to a cycle.
type Recovery struct {
	CycleID    int           `json:"cycle_id"`
	SleepID    string        `json:"sleep_id"` // UUID in v2
	UserID     int           `json:"user_id"`
	CreatedAt  string        `json:"created_at"`
	UpdatedAt  string        `json:"updated_at"`
	ScoreState string        `json:"score_state"`
	Score      RecoveryScore `json:"score"`
}

// SleepNeeded captures sleep debt/need data.
type SleepNeeded struct {
	BaselineMillis          int64 `json:"baseline_milli"`
	NeedFromSleepDebtMillis int64 `json:"need_from_sleep_debt_milli"`
	NeedFromRecentStrainMillis int64 `json:"need_from_recent_strain_milli"`
	NeedFromRecentNapMillis int64 `json:"need_from_recent_nap_milli"`
}

// SleepStageSummary holds stage duration data.
type SleepStageSummary struct {
	TotalInBedTimeMilli         int64 `json:"total_in_bed_time_milli"`
	TotalAwakeTimeMilli         int64 `json:"total_awake_time_milli"`
	TotalNoDataTimeMilli        int64 `json:"total_no_data_time_milli"`
	TotalLightSleepTimeMilli    int64 `json:"total_light_sleep_time_milli"`
	TotalSlowWaveSleepTimeMilli int64 `json:"total_slow_wave_sleep_time_milli"`
	TotalRemSleepTimeMilli      int64 `json:"total_rem_sleep_time_milli"`
	SleepCycleCount             int   `json:"sleep_cycle_count"`
	DisturbanceCount            int   `json:"disturbance_count"`
}

// SleepScore holds sleep scoring data.
type SleepScore struct {
	StageSummary          SleepStageSummary `json:"stage_summary"`
	SleepNeeded           SleepNeeded       `json:"sleep_needed"`
	RespiratoryRate       float64           `json:"respiratory_rate"`
	SleepPerformance      float64           `json:"sleep_performance_percentage"`
	SleepConsistency      float64           `json:"sleep_consistency_percentage"`
	SleepEfficiency       float64           `json:"sleep_efficiency_percentage"`
}

// Sleep represents a WHOOP sleep record.
type Sleep struct {
	ID             string     `json:"id"`      // UUID in v2
	V1ID           *int       `json:"v1_id"`   // deprecated after 09/01/2025, may be nil
	UserID         int        `json:"user_id"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
	Start          string     `json:"start"`
	End            string     `json:"end"`
	TimezoneOffset string     `json:"timezone_offset"`
	Nap            bool       `json:"nap"`
	ScoreState     string     `json:"score_state"`
	Score          SleepScore `json:"score"`
}

// ZoneDuration holds heart rate zone durations for a workout.
type ZoneDuration struct {
	ZoneZeroMillis  int64 `json:"zone_zero_milli"`
	ZoneOneMillis   int64 `json:"zone_one_milli"`
	ZoneTwoMillis   int64 `json:"zone_two_milli"`
	ZoneThreeMillis int64 `json:"zone_three_milli"`
	ZoneFourMillis  int64 `json:"zone_four_milli"`
	ZoneFiveMillis  int64 `json:"zone_five_milli"`
}

// WorkoutScore holds workout scoring data.
type WorkoutScore struct {
	Strain           float64      `json:"strain"`
	AverageHeartRate int          `json:"average_heart_rate"`
	MaxHeartRate     int          `json:"max_heart_rate"`
	Kilojoule        float64      `json:"kilojoule"`
	PercentRecorded  float64      `json:"percent_recorded"`
	DistanceMeter    float64      `json:"distance_meter"`
	AltitudeGainMeter float64     `json:"altitude_gain_meter"`
	AltitudeChangeMeter float64   `json:"altitude_change_meter"`
	ZoneDuration     ZoneDuration `json:"zone_duration"`
}

// Workout represents a WHOOP workout record.
type Workout struct {
	ID             string       `json:"id"`          // UUID in v2
	V1ID           *int         `json:"v1_id"`       // deprecated after 09/01/2025, may be nil
	UserID         int          `json:"user_id"`
	CreatedAt      string       `json:"created_at"`
	UpdatedAt      string       `json:"updated_at"`
	Start          string       `json:"start"`
	End            string       `json:"end"`
	TimezoneOffset string       `json:"timezone_offset"`
	SportID        int          `json:"sport_id"`
	SportName      string       `json:"sport_name"` // new v2 field, preferred over sport_id
	ScoreState     string       `json:"score_state"`
	Score          WorkoutScore `json:"score"`
}

// PaginatedResponse is a generic wrapper for WHOOP paginated API responses.
type PaginatedResponse[T any] struct {
	Records   []T    `json:"records"`
	NextToken string `json:"next_token"`
}

// SPORT_NAMES maps WHOOP sport IDs to human-readable names.
var SPORT_NAMES = map[int]string{
	-1:  "Activity",
	0:   "Running",
	1:   "Cycling",
	16:  "Baseball",
	17:  "Basketball",
	18:  "Rowing",
	19:  "Fencing",
	20:  "Field Hockey",
	21:  "Football",
	22:  "Golf",
	24:  "Ice Hockey",
	25:  "Lacrosse",
	27:  "Rugby",
	28:  "Sailing",
	29:  "Skiing",
	30:  "Soccer",
	31:  "Softball",
	32:  "Squash",
	33:  "Swimming",
	34:  "Tennis",
	35:  "Track & Field",
	36:  "Volleyball",
	37:  "Water Polo",
	38:  "Wrestling",
	39:  "Boxing",
	42:  "Dance",
	43:  "Pilates",
	44:  "Yoga",
	45:  "Weightlifting",
	47:  "Cross Country Skiing",
	48:  "Functional Fitness",
	49:  "Duathlon",
	51:  "Gymnastics",
	52:  "Hiking/Rucking",
	53:  "Horseback Riding",
	55:  "Kayaking",
	56:  "Martial Arts",
	57:  "Mountain Biking",
	59:  "Powerlifting",
	60:  "Rock Climbing",
	61:  "Paddleboarding",
	62:  "Triathlon",
	63:  "Walking",
	64:  "Surfing",
	65:  "Elliptical",
	66:  "Stairmaster",
	70:  "Meditation",
	71:  "Other",
	73:  "Diving",
	74:  "Operations - Tactical",
	75:  "Operations - Medical",
	76:  "Operations - Flying",
	77:  "Operations - Water",
	82:  "Ultimate",
	83:  "Climber",
	84:  "Jumping Rope",
	85:  "Australian Football",
	86:  "Skateboarding",
	87:  "Coaching",
	88:  "Ice Bath",
	89:  "Commuting",
	90:  "Gaming",
	91:  "Snowboarding",
	92:  "Motocross",
	93:  "Cricket",
	94:  "Pickleball",
	95:  "Badminton",
	96:  "Obstacle Course Racing",
	97:  "Motor Racing",
	98:  "HIIT",
	99:  "Spin",
	100: "Jiu Jitsu",
	101: "Manual Labor",
	103: "Archery",
}
