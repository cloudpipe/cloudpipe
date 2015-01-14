package main

const (
	// CodeWTF is returned when an invariant turns out not to be true.
	CodeWTF = "WTF"
	// CodeStorageError means that there was an error interacting with the storage layer.
	CodeStorageError = "STORE"

	// CodeCredentialsMissing means a request that was required to be authenticated had no auth data.
	CodeCredentialsMissing = "ANONE"
	// CodeCredentialsIncorrect means auth data on a request was present, but incorrect.
	CodeCredentialsIncorrect = "AFAIL"
	// CodeAuthServiceConnection means the auth service could not be reached.
	CodeAuthServiceConnection = "ACONN"

	// CodeMethodNotSupported means a request was made against a resource with an unsupported method.
	CodeMethodNotSupported = "MINVAL"
	// CodeUnableToParseQuery means a request contained a malformed query string.
	CodeUnableToParseQuery = "QINVAL"

	// CodeInvalidJobJSON means a POST body to /jobs was not parseable JSON.
	CodeInvalidJobJSON = "JPRS"
	// CodeInvalidJobForm means that a POST body did not contain form-encoded data.
	CodeInvalidJobForm = "JFRM"
	// CodeMissingCommand means a job is missing a "cmd" element.
	CodeMissingCommand = "JCMD"
	// CodeInvalidResultSource means a job has an invalid result source.
	CodeInvalidResultSource = "JRSRC"
	// CodeInvalidResultType means a job has an invalid result type.
	CodeInvalidResultType = "JRTYPE"
	// CodeEnqueueFailure means a job could not be enqueued in the storage engine.
	CodeEnqueueFailure = "JQUEUE"
	// CodeListFailure means that a query for jobs could not be performed by storage engine.
	CodeListFailure = "JLIST"
	// CodeJobKillFailure means that a job's container was unable to be killed.
	CodeJobKillFailure = "JKILL"
	// CodeJobUpdateFailure means that an update to an existing job was unable to be performed.
	CodeJobUpdateFailure = "JUPD"
	// CodeJobNotFound means that an action was attempted on a job that doesn't exist.
	CodeJobNotFound = "JNF"
)
