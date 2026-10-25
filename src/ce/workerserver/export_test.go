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

// SetMasterTasks replaces the master task list registered by RegisterMasterTasks.
func (s *Scheduler) SetMasterTasks(tasks []TaskDefinition) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.masterTaskDefs = tasks
}

// Shutdown stops the underlying scheduler.
func (s *Scheduler) Shutdown() error {
	return s.scheduler.Shutdown()
}
