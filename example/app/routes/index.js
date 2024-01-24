import Route from '@ember/routing/route';

export default class extends Route {
  queryParams = {
    attributes: {
      refreshModel: true,
    },
  };

  model(params) {
    return params;
  }

  afterModel(model) {
    if (model.attributes) {
      document.documentElement.setAttribute('foo', 'html');
      document.head.setAttribute('foo', 'head');
      document.body.setAttribute('foo', 'body');
    }
  }
}
