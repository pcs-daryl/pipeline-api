package log

// PrintMicroserviceHeader prints a default header for any microservice we use.
// This helps to clearly indicate the start of the service and is useful in locating system
// restarts and identifying critical information in a long log printout.
func PrintMicroserviceHeader(serviceName string, serviceVersion string) {
	Info("===========================================================================")
	Infof("%v (%v) STARTED", serviceName, serviceVersion)
	Info("===========================================================================")
}
