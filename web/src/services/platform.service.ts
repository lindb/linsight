import { ApiPath } from '@src/constants';
import { Bootdata } from '@src/types';
import { ApiKit } from '@src/utils';

const boot = (): Promise<Bootdata> => {
  return ApiKit.GET<Bootdata>(ApiPath.Boot);
};

function getProducts() {
  return [
    {
      name: 'Dashboard',
      path: '/dashboard',
      menu: [
        { itemKey: 'dashboard', text: 'Dashboard' },
        { itemKey: 'explore', text: 'Explore' },
        { itemKey: 'metrics', text: 'Metrics' },
      ],
    },
    {
      name: 'APM',
      menu: [
        { itemKey: 'overview', text: 'Overview' },
        { itemKey: 'exception', text: 'Exception' },
      ],
    },
    { name: 'Infrastructure' },
    { name: 'Kubernetes' },
    { name: 'RUM' },
    { name: 'Alert', path: '/alert/alert-list' },
    { name: 'Setting' },
  ];
}

export default {
  boot,
  getProducts,
};
