import { Feature, FeatureRepositoryInst } from '@src/types';
import Explore from '@src/features/explore/Explore';

FeatureRepositoryInst.register(new Feature('/explore', Explore));
