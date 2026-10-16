from locust import LoadTestShape


# Test 6: step load. Kept in its own file because a LoadTestShape in the main
# locustfile would apply to every run and override -u/-r.
# Run with: -f test/locustfile.py,test/shape_step.py ProductStressUser
class StepLoadShape(LoadTestShape):
    stages = [10, 25, 50, 100, 200, 400]  # users per stage; append 800, 1600 if needed
    stage_seconds = 90
    spawn_rate = 50  # users per second when moving to the next stage

    def tick(self):
        stage = int(self.get_run_time() // self.stage_seconds)
        if stage >= len(self.stages):
            return None  # ends the test
        return self.stages[stage], self.spawn_rate
