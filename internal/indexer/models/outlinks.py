import json
from typing import Dict, Any, Set
from dataclasses import dataclass, asdict

@dataclass
class Outlinks:
    _id:    str # page_url
    links:  Set[str]

    def to_dict(self) -> Dict[str, Any]:
        # Convert to dictionary
        data = asdict(self)
        data["links"] = list(self.links)
        return data

    def prettify(self) -> str:
        links_str = "\n" + "\n".join(f"\t│ \t - {link}" for link in self.links) if self.links else "\tNone"
        return f"""
        ┌──────────────────────────────────────────────────────┐
        │ IMAGE URL: {self._id}
        │
        │ OUTLINKS: {links_str}
        └──────────────────────────────────────────────────────┘
        """
