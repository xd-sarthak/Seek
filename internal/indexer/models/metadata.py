import time
from typing import List, Dict, Any
from dataclasses import dataclass, asdict
from datetime import datetime
from email.utils import parsedate_to_datetime, format_datetime
from models.page import Page
from models.metadata import Metadata
from models.outlinks import Outlinks

# COLLECTIONS
WORDS_COLLECTION = "words"
METADATA_COLLECTION = "metadata"
OUTLINKS_COLLECTION = "outlinks"
DICTIONARY_COLLECTION = "dictionary"

@dataclass
class Metadata:
    _id: str
    title:          str
    description:    str
    summary_text:   str
    last_crawled:   str
    keywords:       Dict[str, int] = None

    @classmethod
    def from_dict(cls, metadata: Dict[str, Any]) -> 'Metadata':
        if metadata == None:
            return None

        # Parse fields
        last_crawled = parsedate_to_datetime(metadata['last_crawled'])

        metadata["last_crawled"] = last_crawled
        return cls(**metadata)

    def to_dict(self) -> Dict[str, Any]:
        # Convert to dictionary
        data = asdict(self)
        data["last_crawled"] = self.last_crawled.strftime("%a, %d %b %Y %H:%M:%S ") + time.tzname[0]
        return data

    def prettify(self) -> str:
        return f"""
        -----------------------------------------------------
        URL: {self._id}
        Title: {self.title}
        Description: {self.description[:15] + '...' if len(self.description) > 15 else self.description}
        Summary Text: {self.summary_text[:15] + '...' if len(self.summary_text) > 15 else self.summary_text}
        Last Crawled: {self.last_crawled}
        -----------------------------------------------------
        """
