const config = {
    'NAME': CONFIG
};

window.FastBoot = {
    config(name) {
        return config[name];
    },
    require(name) {
        if (name === 'crypto') {
            return window.crypto;
        }
        if (name === 'node-fetch') {
            return {
                'default': window.fetch,
                FormData: window.FormData,
                Headers: window.Headers,
                Request: window.Request,
                Response: window.Response,
                FetchError: window.FetchError,
                AbortError: window.AbortError,
                isRedirect: window.isRedirect,
                Blob: window.Blob,
                File: window.File,
                fileFromSync: window.fileFromSync,
                fileFrom: window.fileFrom,
                blobFromSync: window.blobFromSync,
                blobFrom: window.blobFrom,
            };
        }
        if (name === 'abortcontroller-polyfill/dist/cjs-ponyfill') {
            return {
                AbortController: window.AbortController,
                AbortSignal: window.AbortSignal,
                fetch: window.fetch,
            };
        }
        return window.require(...arguments);
    },
};
