(async () => {
    HEADERS_CLASS

    if (window.$running) {
        throw new Error('instance running');
    }

    try {
        if (window.$instance) {
            await $instance.destroy();
        }

        window.$running = true;

        let removeAttributes = (node) => {
            while(node.attributes.length > 0) {
                node.removeAttribute(node.attributes[0].name);
            }
        }

        document.head.innerHTML = '';
        document.body.innerHTML = '';
        removeAttributes(document.head);
        removeAttributes(document.body);
        removeAttributes(document.documentElement);

        window.$instance = await $app.buildInstance();

        const request = REQUEST;
        request.headers = new FastBootHeaders(request.headers);
        request.host = () => request.headers.get('host');

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

        $instance.register('info:-fastboot', info, { instantiate: false });

        const options = {
            document: window.document,
            isBrowser: true,
            isInteractive: false,
            rootElement: window.document.body,
        };

        await $instance.boot(options);
        await $instance.visit('URL', options);
        await info.deferredPromise;

        return info;
    } catch (err) {
        throw err;
    } finally {
        window.$running = false;
    }
})()
