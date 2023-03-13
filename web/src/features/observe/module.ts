import { Feature, FeatureRepositoryInst } from '@src/types';
import DynamicDashboard from './DynamicDashboard';

FeatureRepositoryInst.register(new Feature('/observe/*', DynamicDashboard));
