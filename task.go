package pixur

type Task interface {
	/* If there was a retriable error, this will be called before Run */
	ResetForRetry()

	/* Runs cleanup tasks regardless of whether there was a failure or success.
	   This is always run only after all retry attempts are spent
	*/
	CleanUp()

	Run() error
}
