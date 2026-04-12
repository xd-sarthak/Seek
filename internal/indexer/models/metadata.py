from typing import Dict, Any
from dataclasses import dataclass, asdict
from datetime import datetime
from email.utils import parsedate_to_datetime, format_datetime

@dataclass
class Metadata:
    _id: str
    title:          str
    description:    str
    summary_text:   str
    last_crawled:   datetime
    keywords:       Dict[str, int] = None

    @classmethod
    def from_dict(cls, metadata: Dict[str, Any]) -> 'Metadata':
        if metadata is None:
            return None

        # Copy to avoid mutating the caller's dict
        data = dict(metadata)

        # Parse fields
        last_crawled_raw = data.get('last_crawled')
        if last_crawled_raw and isinstance(last_crawled_raw, str):
            data['last_crawled'] = parsedate_to_datetime(last_crawled_raw)

        return cls(**data)

    def to_dict(self) -> Dict[str, Any]:
        # Convert to dictionary
        data = asdict(self)
        if self.last_crawled and hasattr(self.last_crawled, 'strftime'):
            data["last_crawled"] = format_datetime(self.last_crawled)
        return data

    def prettify(self) -> str:
        desc = self.description or ""
        summary = self.summary_text or ""
        return f"""
        -----------------------------------------------------
        URL: {self._id}
        Title: {self.title}
        Description: {desc[:15] + '...' if len(desc) > 15 else desc}
        Summary Text: {summary[:15] + '...' if len(summary) > 15 else summary}
        Last Crawled: {self.last_crawled}
        -----------------------------------------------------
        """
