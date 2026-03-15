import logging
import signal
import os
import sys

from data.redis_client import RedisClient
from data.mongo_client import MongoClient


# setting up the logger
logger = logging.getLogger(__name__)
logging.basicConfig(
    level = logging.INFO,
    format = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

running = True

def handle_exit_signal(signum, frame):
    global running
    logger.info("Received exit signal. Shutting down gracefully...")
    running = False

    logger.info("Closing database connections...")
    # close redis and mongo connections here


    sys.exit(0)


signal.signal(signal.SIGINT, handle_exit_signal)
signal.signal(signal.SIGTERM, handle_exit_signal)


if __name__ == "__main__":

    #redis setup
    redis_host = os.getenv("REDIS_HOST", "localhost")
    redis_port = int(os.getenv("REDIS_PORT", 6379))
    redis_password = os.getenv("REDIS_PASSWORD", None)
    redis_db = int(os.getenv("REDIS_DB", 0))

    #mongo setup
    mongo_host = os.getenv("MONGO_HOST", "localhost")
    mongo_port = int(os.getenv("MONGO_PORT", 27017))
    mongo_password = os.getenv("MONGO_PASSWORD", None)
    mongo_db = os.getenv("MONGO_DB", "indexer")
    mongo_user = os.getenv("MONGO_USER", None)

    #connect to redis
    logger.info(f"Connecting to Redis at {redis_host}:{redis_port}...")
    redis = RedisClient(
        host=redis_host,
        port=redis_port,
        password=redis_password,
        db=redis_db
    )

    if redis.get_client().ping():
        logger.info("Successfully connected to Redis!")
    else:
        logger.error("Failed to connect to Redis. Exiting.")
        sys.exit(1)
    
    #connect to mongo
    logger.info(f"Connecting to MongoDB at {mongo_host}:{mongo_port}...")
    mongo = MongoClient(
        host=mongo_host,
        port=mongo_port,
        username=mongo_user,
        password=mongo_password,
        authSource=mongo_db
    )

    try:
        mongo.admin.command('ping')
        logger.info("Successfully connected to MongoDB!")
    except Exception as e:
        logger.error(f"Failed to connect to MongoDB: {e}. Exiting.")
        sys.exit(1)

    



        # Define thresholds for batch operations
    WORDS_OP_THRESHOLD = 1000
    METADATA_OP_THRESHOLD = 100
    OUTLINKS_OP_THRESHOLD = 100

    # Initialize operation buffers
    create_words_entry_operations = []
    create_metadata_operations = []
    create_outlinks_operations = []

    # Function to perform bulk operations when thresholds are met
    def perform_bulk_operations():
        global create_words_entry_operations, create_metadata_operations, create_outlinks_operations

        if len(create_words_entry_operations) >= WORDS_OP_THRESHOLD:
            logger.info("Performing words bulk operations...")
            mongo.create_words_bulk(create_words_entry_operations)
            create_words_entry_operations = []

        if len(create_metadata_operations) >= METADATA_OP_THRESHOLD:
            logger.info("Performing metadata bulk operations...")
            mongo.create_metadata_bulk(create_metadata_operations)
            create_metadata_operations = []

        if len(create_outlinks_operations) >= OUTLINKS_OP_THRESHOLD:
            logger.info("Performing outlinks bulk operations...")
            mongo.create_outlinks_bulk(create_outlinks_operations)
            create_outlinks_operations = []
