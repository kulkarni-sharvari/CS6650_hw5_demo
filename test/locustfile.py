import random
from locust import HttpUser, FastHttpUser, task, between, TaskSet

class ProductTasks(TaskSet):
    
    @task(9)
    def get_product(self):
        product_id = random.choice([1,2,3])
        self.client.get(f"/products/{product_id}", name="GET productById")
    
    @task(1)
    def post_product_details(self):
        product_id = random.choice([1,2,3])
        cat_id = random.randrange(1,100,1)
        some_other_id = random.randrange(1,100,1)
        payload = {
            "sku": f"SKU-{product_id}-ABC",
            "manufacturer": "Manufacturer A",
            "category_id": cat_id,
            "weight": 50,
            "some_other_id": some_other_id
        }
        self.client.post(f"/products/{product_id}/details", json=payload, name="POST updateProductDetailById")


class ProductHttpUser(HttpUser):
    tasks = [ProductTasks]


class ProductFastHttpUser(FastHttpUser):
    tasks = [ProductTasks]


# Test 6 (step load): FastHttpUser with no wait_time, so each user sends its next
# request as soon as the previous one returns. The number of users over time is
# set by StepLoadShape in shape_step.py, not by -u/-r.
class ProductStressUser(FastHttpUser):
    tasks = [ProductTasks]


# Part V (observability): a mix that includes a few expected errors, so the
# error panel in Grafana and the WARN/ERROR lines in the logs have something to
# show. Only used against the local docker-compose stack.
#   80% GET existing product -> 200
#   10% POST valid update    -> 204
#    8% GET missing product  -> 404 (expected)
#    2% GET with X-Force-Error: true -> 500 (expected, test hook)
class ObservabilityTasks(TaskSet):

    @task(80)
    def get_product(self):
        ProductTasks.get_product(self)

    @task(10)
    def post_product_details(self):
        ProductTasks.post_product_details(self)

    # catch_response lets us mark an expected 4xx/5xx as a success, so Locust
    # does not count these deliberate errors as failures.
    @task(8)
    def get_missing_product(self):
        with self.client.get("/products/999", name="GET productById (missing)",
                             catch_response=True) as response:
            if response.status_code == 404:
                response.success()
            else:
                response.failure(f"expected 404, got {response.status_code}")

    @task(2)
    def get_forced_error(self):
        with self.client.get("/products/1", headers={"X-Force-Error": "true"},
                             name="GET productById (forced 500)",
                             catch_response=True) as response:
            if response.status_code == 500:
                response.success()
            else:
                response.failure(f"expected 500, got {response.status_code}")


class ProductObservabilityUser(FastHttpUser):
    wait_time = between(0.05, 0.2)
    tasks = [ObservabilityTasks]
