import nltk

_initialized = False
STOP_WORDS = None


def initialize_nlp():
    global _initialized, STOP_WORDS

    if not _initialized:
        resources = {
            "punkt": "tokenizers/punkt",
            "punkt_tab": "tokenizers/punkt_tab",
            "stopwords": "corpora/stopwords",
        }
        for resource, resource_path in resources.items():
            try:
                nltk.data.find(resource_path)
            except LookupError:
                nltk.download(resource, quiet=True)
        STOP_WORDS = set(nltk.corpus.stopwords.words('english'))
        _initialized = True

    return STOP_WORDS
