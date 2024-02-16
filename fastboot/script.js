(function () {
    /* Utilities */

    // adapted from https://github.com/ember-fastboot/ember-cli-fastboot/blob/master/packages/fastboot/src/fastboot-headers.js

    class FastBootHeaders {
        headers = {};

        constructor(headers) {
            headers = headers || {};
            for (let header in headers) {
                let value = headers[header];
                if (typeof value === 'string') {
                    value = [value];
                }
                this.headers[header.toLowerCase()] = value;
            }
        }

        append(header, value) {
            header = header.toLowerCase();
            if (!this.has(header)) {
                this.headers[header] = [];
            }
            this.headers[header].push(value);
        }

        delete(header) {
            delete this.headers[header.toLowerCase()];
        }

        entries() {
            let entries = [];
            for (let key in this.headers) {
                let values = this.headers[key];
                for (let index = 0; index < values.length; ++index) {
                    entries.push([key, values[index]]);
                }
            }
            return entries[Symbol.iterator]();
        }

        get(header) {
            return this.getAll(header)[0] || null;
        }

        getAll(header) {
            return this.headers[header.toLowerCase()] || [];
        }

        has(header) {
            return this.headers[header.toLowerCase()] !== undefined;
        }

        keys() {
            let entries = [];
            for (let key in this.headers) {
                let values = this.headers[key];
                for (let index = 0; index < values.length; ++index) {
                    entries.push(key);
                }
            }
            return entries[Symbol.iterator]();
        }

        set(header, value) {
            header = header.toLowerCase();
            this.headers[header] = [value];
        }

        values() {
            let entries = [];
            for (let key in this.headers) {
                let values = this.headers[key];
                for (let index = 0; index < values.length; ++index) {
                    entries.push(values[index]);
                }
            }
            return entries[Symbol.iterator]();
        }

        unknownProperty() {
            throw new Error('FastBootHeaders does not support "unknownProperty" operations.');
        }
    }

    function removeAttributes(node) {
        while(node.attributes.length > 0) {
            node.removeAttribute(node.attributes[0].name);
        }
    }

    /* Functions */

    window.$setup = async function (name, config) {
        window.FastBoot = {
            config(_name) {
                if (_name !== name) {
                    throw new Error('name mismatch: ' + _name);
                }
                return config;
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
    }

    window.$boot = async function () {
        window.$app = require('~fastboot/app-factory').default()
        await $app.boot();
    }

    window.$render = async function (url, request) {
        // abort if running
        if (window.$running) {
            throw new Error('instance running');
        }

        try {
            // destroy existing instance
            if (window.$instance) {
                await $instance.destroy();
            }

            // set flag
            window.$running = true;

            // clear document
            document.head.innerHTML = '';
            document.body.innerHTML = '';
            removeAttributes(document.head);
            removeAttributes(document.body);
            removeAttributes(document.documentElement);

            // build instance
            window.$instance = await $app.buildInstance();

            // setup request
            request.headers = new FastBootHeaders(request.headers);
            request.host = () => request.headers.get('host');

            // setup info
            const info = {
                request: request,
                response: {
                    headers: new FastBootHeaders({}),
                    statusCode: 200,
                },
                metadata: {},
                deferredPromise: Promise.resolve(),
                deferRendering(promise) {
                    this.deferredPromise = promise;
                },
            };

            // register info
            $instance.register('info:-fastboot', info, { instantiate: false });

            // prepare options
            const options = {
                document: window.document,
                isBrowser: true,
                isInteractive: false,
                rootElement: window.document.body,
            };

            // boot and visit
            await $instance.boot(options);
            await $instance.visit(url, options);

            // wait for deferred promise
            await info.deferredPromise;

            return info;
        } catch (err) {
            throw err;
        } finally {
            // clear flag
            window.$running = false;
        }
    }

    window.$capture = function () {
        return {
            headContent: document.head.innerHTML,
            bodyContent: document.body.innerHTML,
            htmlAttributes: Object.fromEntries(Array.from(document.documentElement.attributes).map(a => [a.name, a.value])),
            headAttributes: Object.fromEntries(Array.from(document.head.attributes).map(a => [a.name, a.value])),
            bodyAttributes: Object.fromEntries(Array.from(document.body.attributes).map(a => [a.name, a.value])),
        };
    }
})();
