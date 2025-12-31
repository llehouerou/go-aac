// internal/syntax/pred.go
package syntax

// PredInfo holds MAIN profile prediction data.
// This stores the prediction control information parsed from the bitstream.
//
// Ported from: pred_info in ~/dev/faad2/libfaad/structs.h:201-207
type PredInfo struct {
	// Limit is the maximum SFB for prediction (min of max_sfb and max_pred_sfb)
	Limit uint8

	// PredictorReset indicates if predictors should be reset
	PredictorReset bool

	// PredictorResetGroupNumber is the reset group (1-30), used with modulo 30
	PredictorResetGroupNumber uint8

	// PredictionUsed indicates which SFBs use prediction
	PredictionUsed [MaxSFB]bool
}
