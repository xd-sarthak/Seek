
from datetime import datetime
from typing import List, Dict, Any
from dataclasses import dataclass
from email.utils import parsedate_to_datetime

@dataclass
class Page:
    normalized_url: str
    html: str
    content_type: str
    status_code: int
    last_crawled: datetime

    @classmethod
    def from_hash(cls, page_data: Dict[str, Any]) -> 'Page':

        if page_data == None:
            return None

        # Parse fields
        last_crawled = parsedate_to_datetime(page_data['last_crawled'])

        return cls (
            normalized_url=page_data['normalized_url'],
            html=page_data['html'],
            content_type=page_data['content_type'],
            status_code=int(page_data['status_code']),
            last_crawled=last_crawled,
        )

    def prettify(self) -> str:
        return f"""
        -----------------------------------------------------
        URL: {self.normalized_url}
        HTML: {self.html[:15] + "..." if len(self.html) > 15 else self.html}
        Content Type: {self.content_type}
        Status Code: {self.status_code}
        Last Crawled: {self.last_crawled}
        -----------------------------------------------------
        """

