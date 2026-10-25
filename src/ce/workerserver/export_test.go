package jobs

// JobCount returns the number of jobs registered on the underlying scheduler.
func (s *Scheduler) JobCount() int {
	return len(s.scheduler.Jobs())
}

// MasterTaskCount returns the number of master tasks tracked by the scheduler.
func (s *Scheduler) MasterTaskCount() int {
	s.mux.Lock()
	defer s.mux.Unlock()

	return len(s.masterTasks)
}
