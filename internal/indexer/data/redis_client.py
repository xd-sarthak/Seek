import redis
import logging
import time

#set up logging
logger = logging.getLogger(__name__)
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

class RedisClient:
    def __init__(self, host='localhost', port=6379, password=None, db=0):
        self.host = host
        self.port = port
        self.password = password
        self.db = db
        self.client = None
        self.connect()

    def connect(self):
        try:
            self.client = redis.Redis(
                host=self.host,
                port=self.port,
                password=self.password,
                db=self.db,
                socket_timeout=5  # 5 second timeout for operations
            )
            # Test the connection
            if self.client.ping():
                logger.info(f"Connected to Redis at {self.host}:{self.port} (DB: {self.db})")
            else:
                logger.error(f"Failed to ping Redis at {self.host}:{self.port}")
        except Exception as e:
            logger.error(f"Error connecting to Redis: {e}")
            raise

    def get_client(self):
        return self.client