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
