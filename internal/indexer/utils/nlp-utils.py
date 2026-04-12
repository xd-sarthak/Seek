import nltk

_initialized = False
STOP_WORDS = None

def initialize_nlp():
    global _initialized, STOP_WORDS

    if not _initialized:
        resources = ['punkt', 'stopwords']
        for resource in resources:
            try:
                nltk.data.find(f'corpora/{resource}')
            except LookupError:
                nltk.download(resource)
        STOP_WORDS = set(nltk.corpus.stopwords.words('english'))
        _initialized = True