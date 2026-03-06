import datetime

from locust import TaskSet, FastHttpUser, between
from faker import Faker
import random

# Locust 压测脚本

fake = Faker()

products = [
    '0PUK6V6EV0',
    '1YMWWN1N4O',
    '2ZYFJ3GM2N',
    '66VCHSJNUP',
    '6E92ZMYYFZ',
    '9SIQT8TOJO',
    'L9ECAV7KIM',
    'LS4PSXUNUM',
    'OLJCESPC7Z',
    'MAHOYOSSSS',]

def index(self):
    self.client.get("/")

def setCurrency(self):
    currencies = ['USD', 'JPY', 'CNY', 'HKD']
    self.client.post("/setCurrency",
        {'currency_code': random.choice(currencies)})

def browseProduct(self):
    self.client.get("/product/" + random.choice(products))

def viewCart(self):
    self.client.get("/cart")

def addToCart(self):
    product = random.choice(products)
    self.client.get("/product/" + product)
    self.client.post("/cart", {
        'product_id': product,
        'quantity': random.randint(1,10)})
    
def empty_cart(self):
    self.client.post('/cart/empty')

def checkout(self):
    addToCart(self)
    current_year = datetime.datetime.now().year+1
    self.client.post("/cart/checkout", {
        'email': fake.email(),
        'street_address': fake.street_address(),
        'zip_code': fake.zipcode(),
        'city': fake.city(),
        'state': fake.state_abbr(),
        'country': fake.country(),
        'credit_card_number': fake.credit_card_number(card_type="visa"),
        'credit_card_expiration_month': random.randint(1, 12),
        'credit_card_expiration_year': random.randint(current_year, current_year + 70),
        'credit_card_cvv': f"{random.randint(100, 999)}",
    })
    
class UserBehavior(TaskSet):

    def on_start(self):
        index(self)
    
    tasks = {index: 1,
    setCurrency: 2,
    browseProduct: 10,
    addToCart: 2,
    viewCart: 3,
    checkout: 1}

class WebsiteUser(FastHttpUser):
    tasks = [UserBehavior]
    wait_time = between(1, 10)
