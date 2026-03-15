import time
import logging
import pymongo

#set up logging
logger = logging.getLogger(__name__)
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

class MongoClient:
    def __init__(self, host='localhost', port=27017, username=None, password=None, db_name='indexer'):
        self.host = host
        self.port = port
        self.username = username
        self.password = password
        self.db_name = db_name
        self.client = None
        self.db = None
        self.connect()

    def connect(self):
        try:
            if self.username and self.password:
                uri = f"mongodb://{self.username}:{self.password}@{self.host}:{self.port}/{self.db_name}"
            else:
                uri = f"mongodb://{self.host}:{self.port}/{self.db_name}"
            
            self.client = pymongo.MongoClient(uri, serverSelectionTimeoutMS=5000)
            # Test the connection
            self.client.server_info()  # Will throw an exception if it cannot connect
            self.db = self.client[self.db_name]
            logger.info(f"Connected to MongoDB at {self.host}:{self.port} (DB: {self.db_name})")
        except Exception as e:
            logger.error(f"Error connecting to MongoDB: {e}")
            raise

    def get_db(self):
        return self.db