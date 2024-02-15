import Route from '@ember/routing/route';

export default class extends Route {
  queryParams = {
    timeout: {}
  };

  model(params) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({ message: 'Hello world!' });
      }, parseInt(params.timeout) || 1000);
    });
  }
}
