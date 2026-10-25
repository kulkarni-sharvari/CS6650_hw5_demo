from locust import LoadTestShape


# Exploration: a sudden spike. A quiet start, then the users jump all at once, then they
# stay for the rest of the run. The user count over time comes from this shape, not from
# -u / -r / -t.
# Run with: -f test/locustfile.py,test/shape_spike.py ProductSearchUser
class SpikeShape(LoadTestShape):
    quiet_users = 5
    quiet_seconds = 180      # 3 minutes of light load
    spike_users = 100
    spike_seconds = 600      # then 10 minutes at the spike level
    spawn_rate = 100         # all users start within about a second

    def tick(self):
        t = self.get_run_time()
        if t < self.quiet_seconds:
            return self.quiet_users, self.spawn_rate
        if t < self.quiet_seconds + self.spike_seconds:
            return self.spike_users, self.spawn_rate
        return None  # ends the test
